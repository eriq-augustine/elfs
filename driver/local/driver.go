package local;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "crypto/cipher"
   "io"
   "path/filepath"
   "os"
   "sync"

   "github.com/pkg/errors"

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

func NewDriver(key []byte, iv []byte, path string) (*driver.Driver, error) {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   path, err := filepath.Abs(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to create absolute path for local connector.");
   }

   _, ok := activeConnections[path];
   if (ok) {
      return nil, errors.WithStack(driver.NewIllegalOperationError("Cannot create two connections to the same storage: " + path));
   }

   var connector LocalConnector = LocalConnector {
      path: path,
   };

   return driver.NewDriver(key, iv, &connector);
}

func (this *LocalConnector) PrepareStorage() error {
   return os.MkdirAll(this.path, 0700);
}

func (this *LocalConnector) GetEncryptedReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (io.ReadCloser, error) {
   return newEncryptedFileReader(this.getDiskPath(fileInfo), blockCipher, fileInfo.IV);
}

func (this *LocalConnector) GetMetadataReader(metadataId string, blockCipher cipher.Block, iv []byte) (io.ReadCloser, error) {
   return newEncryptedFileReader(this.getMetadataPath(metadataId), blockCipher, iv);
}

func (this *LocalConnector) GetEncryptedWriter(fileInfo *dirent.Dirent, blockCipher cipher.Block) (io.WriteCloser, error) {
   return newEncryptedFileWriter(this.getDiskPath(fileInfo), blockCipher, fileInfo.IV);
}

func (this *LocalConnector) GetMetadataWriter(metadataId string, blockCipher cipher.Block, iv []byte) (io.WriteCloser, error) {
   return newEncryptedFileWriter(this.getMetadataPath(metadataId), blockCipher, iv);
}

// A convenience function for synchronious writes.
func (this *LocalConnector) Write(fileInfo *dirent.Dirent, blockCipher cipher.Block, clearbytes io.Reader) (uint64, string, error) {
   writer, err := newEncryptedFileWriter(this.getDiskPath(fileInfo), blockCipher, fileInfo.IV);
   if (err != nil) {
      return 0, "", errors.Wrap(err, "Failed to get encrypted writer.");
   }

   // We will be kind to the writer and give it chunks of the optimal size.
   var data []byte = make([]byte, IO_BLOCK_SIZE);

   var done bool = false;
   for (!done) {
      // Ensure we have the correct length.
      data = data[0:IO_BLOCK_SIZE];

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

func (this* LocalConnector) Close() error {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   activeConnections[this.path] = false;
   return nil;
}
