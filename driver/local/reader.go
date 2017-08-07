package local;

// A reader specifically for encrypted files.

import (
   "crypto/cipher"
   "fmt"
   "io"
   "os"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/util"
)

// A Reader that will read an encrypted file, decrypt them, and return the cleartext
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
      golog.ErrorE("Unable to open file on disk at: " + path, err);
      return nil, err;
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
      return 0, fmt.Errorf("Buffer for encryptedFileReader is too small. Must be at least %d.", IO_BLOCK_SIZE);
   }

   // Resize the buffer (without allocating) to ensure we only read exactly what we want.
   this.buffer = this.buffer[0:IO_BLOCK_SIZE + this.gcm.Overhead()];

   // Get the ciphertext.
   _, err := this.fileReader.Read(this.buffer);
   if (err != nil) {
      if (err != io.EOF) {
         return 0, err;
      }

      this.done = true;
   }

   // Resize the destination so we can reliably check the output size.
   outBuffer = outBuffer[0:0];

   _, err = this.gcm.Open(outBuffer, this.iv, this.buffer, nil);
   if (err != nil) {
      golog.ErrorE("Failed to decrypt file.", err);
      return 0, err;
   }

   // Prepare the IV for the next decrypt.
   util.IncrementBytes(this.iv);

   return len(outBuffer), nil;
}

func (this *encryptedFileReader) Close() error {
   return this.fileReader.Close();
}
