package local;

// A connector that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "crypto/cipher"
   "path/filepath"
   "os"
   "sync"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/cipherio"
   "github.com/eriq-augustine/s3efs/dirent"
)

// TODO(eriq): Lock files.

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

func NewLocalConnector(path string) (*LocalConnector, error) {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   path, err := filepath.Abs(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to create absolute path for local connector.");
   }

   _, ok := activeConnections[path];
   if (ok) {
      return nil, errors.Errorf("Cannot create two connections to the same storage: %s", path);
   }

   var connector LocalConnector = LocalConnector {
      path: path,
   };

   return &connector, nil;
}

func (this *LocalConnector) GetId() string {
   return "Local:" + this.path;
}

func (this *LocalConnector) PrepareStorage() error {
   return os.MkdirAll(this.path, 0700);
}

func (this *LocalConnector) GetCipherReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (*cipherio.CipherReader, error) {
   var path string = this.getDiskPath(fileInfo);

   file, err := os.Open(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to open file on disk at: " + path);
   }

   return cipherio.NewCipherReader(file, blockCipher, fileInfo.IV);
}

func (this *LocalConnector) GetMetadataReader(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherReader, error) {
   var path string = this.getMetadataPath(metadataId);

   file, err := os.Open(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to open file on disk at: " + path);
   }

   return cipherio.NewCipherReader(file, blockCipher, iv);
}

func (this *LocalConnector) GetCipherWriter(fileInfo *dirent.Dirent, blockCipher cipher.Block) (*cipherio.CipherWriter, error) {
   var path string = this.getDiskPath(fileInfo);

   file, err := os.Create(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to create file on disk at: " + path);
   }

   err = file.Chmod(0600);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to change file permissions of: " + path);
   }

   return cipherio.NewCipherWriter(file, blockCipher, fileInfo.IV);
}

func (this *LocalConnector) GetMetadataWriter(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherWriter, error) {
   var path string = this.getMetadataPath(metadataId);

   file, err := os.Create(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to create file on disk at: " + path);
   }

   err = file.Chmod(0600);
   if (err != nil) {
      return nil, errors.Wrap(err, "Unable to change file permissions of: " + path);
   }

   return cipherio.NewCipherWriter(file, blockCipher, iv);
}

func (this* LocalConnector) Close() error {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   activeConnections[this.path] = false;
   return nil;
}
