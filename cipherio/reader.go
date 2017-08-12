package cipherio;

// A reader specifically for encrypted data.
// This will wrap another reader that is expected to deliver encrypted content.
// This will buffer any content necessary to get enough data to decrypt chunks.

import (
   "crypto/cipher"
   "io"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/util"
)

// A ReadCloser that will read an encrypted file, decrypt them, and return the cleartext
// all in chunks of size IO_BLOCK_SIZE.
// Note that the cleartext will be in checks of IO_BLOCK_SIZE,
// but the cipertext read will be slightly larger.
type CipherReader struct {
   gcm cipher.AEAD
   ciphertextBuffer []byte
   // We will always read from disk in chunks of IO_BLOCK_SIZE (+ cipher overhead).
   // So, we will need to keep a buffer on hand of what we have read from disk that the reader has not 
   // yet requested.
   cleartextBuffer []byte
   // We need to keep the original slice around so we can resize without reallocating.
   // We will be reslicing the cleartextBuffer as the cleartext is requested.
   originalCleartextBuffer []byte
   iv []byte
   reader io.ReadCloser
   done bool
}

// Caller gives up control of the reader.
func NewCipherReader(reader io.ReadCloser,
      blockCipher cipher.Block, rawIV []byte) (*CipherReader, error) {
   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, err;
   }

   var cleartextBuffer []byte = make([]byte, 0, IO_BLOCK_SIZE);

   var rtn CipherReader = CipherReader{
      gcm: gcm,
      // Allocate enough room for the ciphertext.
      ciphertextBuffer: make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead()),
      cleartextBuffer: cleartextBuffer,
      originalCleartextBuffer: cleartextBuffer,
      // Make a copy of the IV since we will be incrementing it for each chunk.
      iv: append([]byte(nil), rawIV...),
      reader: reader,
      done: false,
   };

   return &rtn, nil;
}

func (this *CipherReader) Read(outBuffer []byte) (int, error) {
   if (this.done) {
      return 0, io.EOF;
   }

   // Keep track of the original output buffer so we can calculate final size correctly.
   var originalOutBuffer []byte = outBuffer;

   // We will keep reading until there is no more to read or the buffer is full.
   for (len(outBuffer) > 0) {
      // First check to see if we have data already buffered.
      if (len(this.cleartextBuffer) != 0) {
         var copyLength int = util.MinInt(len(this.cleartextBuffer), len(outBuffer));
         copy(outBuffer, this.cleartextBuffer[0:copyLength]);

         // Reslice the cleantext buffer and outBuffers to show the copy.
         outBuffer = outBuffer[copyLength:];
         this.cleartextBuffer = this.cleartextBuffer[copyLength:];

         // Reset the cleartext buffer if necessary
         if (len(this.cleartextBuffer) == 0) {
            this.cleartextBuffer = this.originalCleartextBuffer;
         }

         if (len(outBuffer) == 0) {
            return len(originalOutBuffer), nil;
         }
      }

      if (!this.done) {
         // Now read more data into cleartext buffer.
         err := this.readChunk();
         if (err != nil) {
            return 0, errors.Wrap(err, "Failed to read chunk");
         }
      }

      // If we have reached an EOF and there is nothing left in the cleartext buffer,
      // than we have read everything, but fell short of the requested amount.
      if (this.done && len(this.cleartextBuffer) == 0) {
         return (len(originalOutBuffer) - len(outBuffer)), io.EOF;
      }
   }

   // The output buffer is filled and there is something left in the cleartext buffer.
   return len(originalOutBuffer), nil;
}


func (this *CipherReader) readChunk() error {
   // The cleartext buffer better be totally used (empty).
   if (len(this.cleartextBuffer) != 0) {
      return errors.New("Cleartext buffer is not empty.");
   }

   // Resize the buffer (without allocating) to ensure we only read exactly what we want.
   this.ciphertextBuffer = this.ciphertextBuffer[0:IO_BLOCK_SIZE + this.gcm.Overhead()];

   // Get the ciphertext.
   readSize, err := this.reader.Read(this.ciphertextBuffer);
   if (err != nil) {
      if (err != io.EOF) {
         return errors.Wrap(err, "Failed to read ciphertext chunk");
      }

      this.done = true;
   }

   if (readSize == 0) {
      return nil;
   }

   // Reset the clear text buffer.
   this.cleartextBuffer = this.originalCleartextBuffer;

   // Reuse the outBuffer's memory.
   this.cleartextBuffer, err = this.gcm.Open(this.cleartextBuffer, this.iv, this.ciphertextBuffer[0:readSize], nil);
   if (err != nil) {
      return errors.Wrap(err, "Failed to decrypt chunk");
   }

   // Prepare the IV for the next decrypt.
   util.IncrementBytes(this.iv);

   return nil;
}

func (this *CipherReader) Close() error {
   return this.reader.Close();
}
