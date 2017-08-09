package local;

// Not actually an io.Writer, just a function for
// encrypting and writing.
// Note that this does not update any metadata.

import (
   "crypto/cipher"
   "crypto/md5"
   "fmt"
   "hash"
   "os"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/util"
)

// A WriteCloser that will write an encrypted file from cleartext bytes.
// All writes to the actual storage made by this writer will be in chunks of
// IO_BLOCK_SIZE (+ the overhead for cipertext).
// It is possible to write everything at once, but this writer is really meant
// to be streamed in smaller (closer to IO_BLOCK_SIZE) chunks.
// Close() MUST BE CALLED after all reading is finished.
// Without the Close() call, the final chunk will not get writen.
// The file size (cleartext) and md5 will be available after the writer is closed.
type encryptedFileWriter struct {
   gcm cipher.AEAD
   // We need to keep the original slice around so we can resize without reallocating.
   // We will be reslicing the cleartextBuffer so we can encrypt in chunks.
   // Any remaining aount will need to be moved back to the beginning,
   // but without a copy we do not know where the beginning of the array is.
   originalCleartextBuffer []byte
   cleartextBuffer []byte
   ciphertextBuffer []byte
   iv []byte
   fileWriter *os.File
   done bool
   fileSize uint64
   md5Hash hash.Hash
}

func (this *encryptedFileWriter) GetFileSize() uint64 {
   if (!this.done) {
      panic("Can't get the filesize of an open encryptedFileWriter");
   }

   return this.fileSize;
}

// Get the md5 as a hex string.
func (this *encryptedFileWriter) GetHash() string {
   if (!this.done) {
      panic("Can't get the hash of an open encryptedFileWriter");
   }

   return fmt.Sprintf("%x", this.md5Hash.Sum(nil));
}

func newEncryptedFileWriter(path string,
      blockCipher cipher.Block, rawIV []byte) (*encryptedFileWriter, error) {
   // TODO(eriq): Do we need to create a different GCM (AEAD) every time?
   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, err;
   }

   fileWriter, err := os.Create(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to create file on disk at: " + path);
   }

   err = fileWriter.Chmod(0600);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to change file permissions of: " + path);
   }

   var cleartextBuffer []byte = make([]byte, 0, IO_BLOCK_SIZE);

   var rtn encryptedFileWriter = encryptedFileWriter{
      gcm: gcm,
      originalCleartextBuffer: cleartextBuffer,
      cleartextBuffer: cleartextBuffer,
      // Allocate enough room for the ciphertext.
      ciphertextBuffer: make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead()),
      // Make a copy of the IV since we will be incrementing it for each chunk.
      iv: append([]byte(nil), rawIV...),
      fileWriter: fileWriter,
      done: false,
      fileSize: 0,
      md5Hash: md5.New(),
   };

   return &rtn, nil;
}

func (this *encryptedFileWriter) Write(data []byte) (int, error) {
   // Grow our local cleartext buffer
   this.cleartextBuffer = append(this.cleartextBuffer, data...);

   // Now just write any available chunks.
   err := this.writeChunks();
   if (err != nil) {
      return 0, errors.Wrap(err, "Failed to write chunks");
   }

   // Unless we have an error, just claim all the data was written.
   return len(data), nil;
}

func (this *encryptedFileWriter) writeChunks() error {
   // We don't have enough to write yet.
   if (len(this.cleartextBuffer) < IO_BLOCK_SIZE && !this.done) {
      return nil;
   }

   // Keep writing as many chunks as we have data for.
   // If we are done, then write the final chunk.
   for (len(this.cleartextBuffer) >= IO_BLOCK_SIZE || (this.done && len(this.cleartextBuffer) > 0)) {
      var writeSize int = IO_BLOCK_SIZE;
      if (len(this.cleartextBuffer) < IO_BLOCK_SIZE) {
         writeSize = len(this.cleartextBuffer);
      }
      var data []byte = this.cleartextBuffer[0:writeSize];

      // Resise the clear text buffer so we "consume" what we are currently writing.
      this.cleartextBuffer = this.cleartextBuffer[writeSize:];

      this.fileSize += uint64(writeSize);
      this.md5Hash.Write(data);

      // Use the shared buffer's memory.
      cipherText := this.gcm.Seal(this.ciphertextBuffer, this.iv, data, nil);

      _, err := this.fileWriter.Write(cipherText);
      if (err != nil) {
         return errors.Wrap(err, "Failed to write file block");
      }

      // Prepare the IV for the next encrypt.
      util.IncrementBytes(this.iv);
   }

   // Move any remaining data to the front of the original buffer.
   copy(this.originalCleartextBuffer, this.cleartextBuffer);
   this.cleartextBuffer = this.originalCleartextBuffer
   this.originalCleartextBuffer = this.originalCleartextBuffer[:0];

   return nil;
}

func (this *encryptedFileWriter) Close() error {
   this.done = true;
   err := this.writeChunks();
   if (err != nil) {
      return errors.Wrap(err, "Failed final write");
   }

   return this.fileWriter.Close();
}
