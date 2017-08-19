package connector;

import (
   "crypto/cipher"
   "io"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/dirent"
)

// A convenience function for synchronious writes.
// Returns: (size, md5 hash (hex), error).
func Write(connector Connector, fileInfo *dirent.Dirent,
      blockCipher cipher.Block, clearbytes io.Reader) (uint64, string, error) {
   writer, err := connector.GetCipherWriter(fileInfo, blockCipher);
   if (err != nil) {
      return 0, "", errors.Wrap(err, "Failed to get writer from connector.");
   }

   // We will be kind to the writer and give it chunks of the optimal size.
   var data []byte = make([]byte, cipherio.IO_BLOCK_SIZE);

   var done bool = false;
   for (!done) {
      // Ensure we have the correct length.
      data = data[0:cipherio.IO_BLOCK_SIZE];

      readSize, err := clearbytes.Read(data);
      if (err != nil) {
         if (err != io.EOF) {
            return 0, "", errors.Wrap(err, "Failed to read clearbytes.");
         }

         done = true;
      }

      if (readSize > 0) {
         _, err = writer.Write(data[0:readSize]);
         if (err != nil) {
            return 0, "", errors.Wrap(err, "Failed to write.");
         }
      }
   }

   err = writer.Close();
   if (err != nil) {
      return 0, "", errors.Wrap(err, "Failed to close the writer.");
   }

   return writer.GetFileSize(), writer.GetHash(), nil;
}
