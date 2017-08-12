package driver;

// The driver is responsible for handleing all the filesystem operations.
// A driver will have a connector that will handle the operations to the actual backend
// (eg local filesystem or S3).

import (
   "crypto/aes"
   "crypto/cipher"

   "github.com/eriq-augustine/s3efs/connector"
   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

// TODO(eriq): Writes to FAT probably need a lock.

// TODO(eriq): Need to async operations and keep track of what files currently have read or writes.

type Driver struct {
   connector connector.Connector
   blockCipher cipher.Block
   // IV for metadata tables.
   iv []byte
   fat map[dirent.Id]*dirent.Dirent
   users map[user.Id]*user.User
   groups map[group.Id]*group.Group
   // A map of all directories to their children.
   dirs map[dirent.Id][]*dirent.Dirent
}

// Get a new, uninitialized driver.
// Normally you will want to get a storage specific driver, like a NewLocalDriver.
// If you need a new filesystem, you should call CreateFilesystem().
// If you want to load up an existing filesystem, then you should call SyncFromDisk().
func newDriver(key []byte, iv []byte, connector connector.Connector) (*Driver, error) {
   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      return nil, err;
   }

   var driver Driver = Driver{
      connector: connector,
      blockCipher: blockCipher,
      iv: iv,
      fat: make(map[dirent.Id]*dirent.Dirent),
      users: make(map[user.Id]*user.User),
      groups: make(map[group.Id]*group.Group),
      dirs: make(map[dirent.Id][]*dirent.Dirent),
   };

   return &driver, nil;

   // TODO(eriq): Load cache and write any changes to the disk FAT.
}
