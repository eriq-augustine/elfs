package local;

// Not actually an io.Writer, just a function for
// encrypting and writing.
// Note that this does not update any metadata.

import (
   "crypto/cipher"
   "crypto/md5"
   "fmt"
   "hash"
   "io"
   "os"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/util"
)

// Write some general cleartext to disk.
// All the metadata management will be left out since we could be writing the
// FAT which itself does not have any metadata.
// Returns: (file size, md5 hash (hex string), error).
func (this *LocalDriver) write(clearbytes io.Reader, rawIV []byte, path string) (uint64, string, error) {
   // TODO(eriq): Do we need to create a different GCM (AEAD) every time?
   gcm, err := cipher.NewGCM(this.blockCipher);
   if err != nil {
      return 0, "", err;
   }

   fileWriter, err := os.Create(path);
   if (err != nil) {
      golog.ErrorE("Unable to create file on disk at: " + path, err);
      return 0, "", err;
   }
   defer fileWriter.Close();

   err = fileWriter.Chmod(0600);
   if (err != nil) {
      golog.ErrorE("Unable to change file permissions of: " + path, err);
      return 0, "", err;
   }

   // Make a copy of the IV since we will be incrementing it for each chunk.
   var iv []byte = append([]byte(nil), rawIV...);

   // Allocate enough room for the cleatext and ciphertext.
   var buffer []byte = make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead());
   var fileSize uint64 = 0;
   var m5dHash hash.Hash = md5.New();

   var done bool = false;
   for (!done) {
      // Always resize (not reallocate) to the block size.
      readSize, err := clearbytes.Read(buffer[0:IO_BLOCK_SIZE]);
      if (err != nil) {
         if (err == io.EOF) {
            done = true;
         } else {
            return 0, "", err;
         }
      }

      // Keep track of the size and hash.
      fileSize += uint64(readSize);
      m5dHash.Write(buffer);

      if (readSize > 0) {
         // Reuse the buffer for the cipertext.
         gcm.Seal(buffer[:0], iv, buffer[0:readSize], nil);
         _, err := fileWriter.Write(buffer);
         if (err != nil) {
            golog.ErrorE("Failed to write file block for: " + path, err);
            return 0, "", err;
         }

         // Prepare the IV for the next encrypt.
         util.IncrementBytes(iv);
      }
   }

   return fileSize, fmt.Sprintf("%x", m5dHash.Sum(nil)), nil;
}
