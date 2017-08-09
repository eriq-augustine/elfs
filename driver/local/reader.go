package local;

// A reader specifically for encrypted files.

import (
   "crypto/cipher"
   "io"
   "os"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/util"
)

// A ReadCloser that will read an encrypted file, decrypt them, and return the cleartext
// all in chunks of size IO_BLOCK_SIZE.
// Note that the cleartext will be in checks of IO_BLOCK_SIZE,
// but the cipertext read will be slightly larger.
type encryptedFileReader struct {
   gcm cipher.AEAD
   buffer []byte
   iv []byte
   fileReader *os.File
   done bool
}

func newEncryptedFileReader(path string,
      blockCipher cipher.Block, rawIV []byte) (*encryptedFileReader, error) {
   // TODO(eriq): Do we need to create a different GCM (AEAD) every time?
   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, err;
   }

   fileReader, err := os.Open(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to open file on disk at: " + path);
   }

   var rtn encryptedFileReader = encryptedFileReader{
      gcm: gcm,
      // Allocate enough room for the ciphertext.
      buffer: make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead()),
      // Make a copy of the IV since we will be incrementing it for each chunk.
      iv: append([]byte(nil), rawIV...),
      fileReader: fileReader,
      done: false,
   };

   return &rtn, nil;
}

func (this *encryptedFileReader) Read(outBuffer []byte) (int, error) {
   if (this.done) {
      return 0, io.EOF;
   }

   if (cap(outBuffer) < IO_BLOCK_SIZE) {
      return 0, errors.Errorf("Buffer for encryptedFileReader is too small. Must be at least %d.", IO_BLOCK_SIZE);
   }

   // Resize the buffer (without allocating) to ensure we only read exactly what we want.
   this.buffer = this.buffer[0:IO_BLOCK_SIZE + this.gcm.Overhead()];

   // Get the ciphertext.
   readSize, err := this.fileReader.Read(this.buffer);
   if (err != nil) {
      if (err != io.EOF) {
         return 0, err;
      }

      this.done = true;
   }

   if (readSize == 0) {
      return 0, io.EOF;
   }

   // Reuse the outBuffer's memory.
   outBuffer, err = this.gcm.Open(outBuffer[:0], this.iv, this.buffer[0:readSize], nil);
   if (err != nil) {
      errors.Wrap(err, "Failed to decrypt file.");
      return 0, err;
   }

   // Prepare the IV for the next decrypt.
   util.IncrementBytes(this.iv);

   return len(outBuffer), nil;
}

func (this *encryptedFileReader) Close() error {
   return this.fileReader.Close();
}
