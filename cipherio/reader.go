package cipherio;

// A reader specifically for encrypted data.
// This will wrap another reader that is expected to deliver encrypted content.
// This will buffer any content necessary to get enough data to decrypt chunks.

import (
   "crypto/cipher"
   "io"
   "syscall"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/util"
)

// A ReadSeekCloser that will read an encrypted file, decrypt them, and return the cleartext
// all in chunks of size IO_BLOCK_SIZE.
// Note that the cleartext will be in chunks of IO_BLOCK_SIZE,
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

   // Since we may seek, we need to keep around the original IV.
   iv []byte
   originalIV []byte

   reader util.ReadSeekCloser

   // These are NOT offsets into the respective buffers,
   // there are absolute offsets into the cipher/clear text files.
   // Note that one cannot always be translated into the other using the bocksize and cipher overhead.
   // This is because the ciphertextOffset is the offset into the file on disk.
   // The cleartextOffset, however, is the offset into the bytes we have passed the user.
   // So the cipher offset may be further along (after blocksize translation) because we may have 
   // data in the cleartext buffer.
   ciphertextOffset int64
   cleartextOffset int64

   cleartextSize int64

   cipherBlockSize int64
}

// Caller gives up control of the reader.
func NewCipherReader(reader util.ReadSeekCloser,
      blockCipher cipher.Block, rawIV []byte,
      ciphertextSize int64) (util.ReadSeekCloser, error) {
   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, errors.WithStack(err);
   }

   var cleartextBuffer []byte = make([]byte, 0, IO_BLOCK_SIZE);

   var cipherBlockSize int64 = int64(IO_BLOCK_SIZE + gcm.Overhead());
   // Note that this is exact since we can't write partial blocks.
   var numBlocks int64 = ciphertextSize / cipherBlockSize;
   var cleartextSize int64 = int64(numBlocks * IO_BLOCK_SIZE) + (ciphertextSize - (numBlocks * cipherBlockSize) - int64(gcm.Overhead()));

   var rtn CipherReader = CipherReader{
      gcm: gcm,
      // Allocate enough room for the ciphertext.
      ciphertextBuffer: make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead()),
      cleartextBuffer: cleartextBuffer,
      originalCleartextBuffer: cleartextBuffer,
      // Make a copy of the IV since we will be incrementing it for each chunk.
      iv: append([]byte(nil), rawIV...),
      originalIV: append([]byte(nil), rawIV...),
      reader: reader,
      ciphertextOffset: 0,
      cleartextOffset: 0,
      cleartextSize: cleartextSize,
      cipherBlockSize: cipherBlockSize,
   };

   return &rtn, nil;
}

func (this *CipherReader) Read(outBuffer []byte) (int, error) {
   // We are done if we are staring at the end of the file.
   // Note that we may seek back and read more.
   if (this.cleartextOffset >= this.cleartextSize) {
      return 0, io.EOF;
   }

   // Keep track of the offset when we started this read so we can calculate final read size correctly.
   var originalCleartextOffset = this.cleartextOffset;

   // We will keep reading until there is no more to read or the buffer is full.
   // We will return insize the loop with an EOF if there is no more to read.
   for (len(outBuffer) > 0) {
      // First, make sure that we have data in the cleartext buffer.
      err := this.populateCleartextBuffer();
      if (err != nil) {
         return 0, errors.WithStack(err);
      }

      // Figure out how much to read from the buffer (min of room left for output and avaible in cleartext).
      var copyLength int = util.MinInt(len(this.cleartextBuffer), len(outBuffer));
      copy(outBuffer, this.cleartextBuffer[0:copyLength]);

      // Reslice the cleartext buffer and outBuffers to show the copy.
      outBuffer = outBuffer[copyLength:];
      this.cleartextBuffer = this.cleartextBuffer[copyLength:];

      // Note the copy in the cleartext offset.
      this.cleartextOffset += int64(copyLength);

      // Reset the cleartext buffer if necessary
      if (len(this.cleartextBuffer) == 0) {
         this.cleartextBuffer = this.originalCleartextBuffer;
      }

      // If we have reached an EOF then we have read everything possible,
      // and either fell short of the requested amount or got that amount exactly.
      // Note that we are checking the cleartext offset instead of the ciphertext offset because
      // the cleartext offset indicates that there is nothing left in the cleartext buffer.
      if (this.cleartextOffset >= this.cleartextSize) {
         return int(this.cleartextOffset - originalCleartextOffset), io.EOF;
      }
   }

   // The output buffer is filled and we have not reached the end of the file.
   return int(this.cleartextOffset - originalCleartextOffset), nil;
}

// Make sure that there is data in the cleartext buffer.
// If there is, then just return.
func (this *CipherReader) populateCleartextBuffer() error {
   if (len(this.cleartextBuffer) != 0) {
      return nil;
   }

   return errors.WithStack(this.readChunk());
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
   }

   if (readSize == 0) {
      return nil;
   }

   // Move the cipher offset forward.
   this.ciphertextOffset += int64(readSize);

   // Reset the cleartext buffer.
   this.cleartextBuffer = this.originalCleartextBuffer;

   this.cleartextBuffer, err = this.gcm.Open(this.cleartextBuffer, this.iv, this.ciphertextBuffer[0:readSize], nil);
   if (err != nil) {
      return errors.Wrap(err, "Failed to decrypt chunk");
   }

   // Prepare the IV for the next decrypt.
   util.IncrementBytes(this.iv);

   return nil;
}

func (this *CipherReader) Seek(offset int64, whence int) (int64, error) {
   absoluteOffset, err := this.absoluteSeekOffset(offset, whence);
   if (err != nil) {
      return this.cleartextOffset, errors.WithStack(err);
   }

   // It is not strange to Seek(io.SeekCurrent, 0) just to see where the reader is.
   if (absoluteOffset == this.cleartextOffset) {
      return this.cleartextOffset, nil;
   }

   // Clear all the buffers and set the offsets to 0.
   // It is possible that we only need to seek a but within the current buffer,
   // but it is easier to just treat all casses the same.
   this.cleartextBuffer = this.originalCleartextBuffer;
   this.ciphertextBuffer = this.ciphertextBuffer[0:IO_BLOCK_SIZE + this.gcm.Overhead()];
   this.iv = append([]byte(nil), this.originalIV...);
   this.ciphertextOffset = 0;
   this.cleartextOffset = 0;
   this.reader.Seek(0, io.SeekStart);

   // Skip the required number of blocks.
   var skipBlocks int64 = absoluteOffset / IO_BLOCK_SIZE;
   util.IncrementBytesByCount(this.iv, int(skipBlocks));
   this.ciphertextOffset = skipBlocks * this.cipherBlockSize;
   this.reader.Seek(this.ciphertextOffset, io.SeekStart);

   // Now read the current block and set the cleartext buffer and offset accordingly.
   err = this.readChunk();
   if (err != nil) {
      // If we fail a read at this point, it is pretty un-recoverable.
      return 0, errors.WithStack(err);
   }

   // The cleartext buffer should be filled, so reslice the buffer to the offset.
   var bufferOffset int = int(absoluteOffset - (skipBlocks * IO_BLOCK_SIZE));
   this.cleartextBuffer = this.cleartextBuffer[bufferOffset:];

   // Finally, we can change the cleartext offset.
   this.cleartextOffset = absoluteOffset;

   return this.cleartextOffset, nil;
}

// Deall with all the different wences and give the absolute offset from the start of the file.
// If the seek offset is not valid in any way, a corresponding error will be retutned.
func (this *CipherReader) absoluteSeekOffset(offset int64, whence int) (int64, error) {
   switch whence {
      case io.SeekStart:
         // Nothing to do.
      case io.SeekCurrent:
         offset = this.cleartextOffset + offset;
      case io.SeekEnd:
         offset = this.cleartextSize + offset;
      default:
         return 0, errors.Wrapf(syscall.EINVAL, "Unknown whence for seek: %d", whence);
   }

   if (offset < 0 || offset > this.cleartextSize) {
      return 0, errors.WithStack(syscall.EINVAL);
   }

   return offset, nil;
}

func (this *CipherReader) Close() error {
   this.gcm = nil;
   this.ciphertextBuffer = nil;
   this.cleartextBuffer = nil;
   this.originalCleartextBuffer = nil;
   this.iv = nil;

   err := this.reader.Close();
   this.reader = nil;

   return errors.WithStack(err);
}
