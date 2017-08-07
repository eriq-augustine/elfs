package local;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "crypto/cipher"
   "io"
   "path/filepath"
   "os"
   "sync"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/driver"
)

// TODO(eriq): Lock files.

const (
   // When doing reads or writes, the size of data to work with in bytes.
   IO_BLOCK_SIZE = 1024 * 1024 * 4
)

// Keep track of the active connections so two instances don't connect to the same storage.
var activeConnections map[string]bool;
var activeConnectionsLock *sync.Mutex;

func init() {
   activeConnections = make(map[string]bool);
   activeConnectionsLock = &sync.Mutex{};
}

type LocalConnector struct {
   path string
}

func NewLocalDriver(key []byte, path string) (*driver.Driver, error) {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   path, err := filepath.Abs(path);
   if (err != nil) {
      golog.ErrorE("Failed to create absolute path for local connector.", err);
      return nil, err;
   }

   _, ok := activeConnections[path];
   if (ok) {
      err = driver.NewIllegalOperationError("Cannot create two connections to the same storage: " + path);
      return nil, err;
   }

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

func (this* LocalConnector) Close() error {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   activeConnections[this.path] = false;
   return nil;
}
