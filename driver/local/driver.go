package local;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "crypto/cipher"
   "io"
   "os"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/driver"
)

const (
   // When doing reads or writes, the size of data to work with in bytes.
   IO_BLOCK_SIZE = 1024 * 1024 * 4
)

type LocalConnector struct {
   path string
}

func NewLocalDriver(key []byte, path string) (*driver.Driver, error) {
   var connector LocalConnector = LocalConnector {
      path: path,
   };

   return driver.NewDriver(key, &connector);
}

func (this *LocalConnector) PrepareStorage() error {
   return os.MkdirAll(this.path, 0700);
}

func (this *LocalConnector) GetEncryptedReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (io.ReadCloser, error) {
   return newEncryptedFileReader(this.getDiskPath(fileInfo), blockCipher, fileInfo.IV);
}

func (this *LocalConnector) Write(fileInfo *dirent.Dirent, blockCipher cipher.Block, clearbytes io.Reader) (uint64, string, error) {
   return this.write(this.getDiskPath(fileInfo), blockCipher, fileInfo.IV, clearbytes);
}
