package driver;

// The driver is responsible for handleing all the filesystem operations.
// A driver will have a connector that will handle the operations to the actual backend
// (eg local filesystem or S3).

import (
   "crypto/aes"
   "crypto/cipher"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/cache"
   "github.com/eriq-augustine/s3efs/connector"
   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

type Driver struct {
   connector connector.Connector
   blockCipher cipher.Block
   fat map[dirent.Id]*dirent.Dirent
   users map[user.Id]*user.User
   groups map[group.Id]*group.Group
   cache *cache.MetadataCache
   // A map of all directories to their children.
   dirs map[dirent.Id][]*dirent.Dirent
   // Base IV for metadata tables.
   iv []byte
   // Speific IVs for metadata tables.
   usersIV []byte
   groupsIV []byte
   fatIV []byte
   cacheIV []byte
}

// Get a new, uninitialized driver.
// Normally you will want to get a storage specific driver, like a NewLocalDriver.
// If you need a new filesystem, you should call CreateFilesystem().
// If you want to load up an existing filesystem, then you should call SyncFromDisk().
func newDriver(key []byte, iv []byte, connector connector.Connector) (*Driver, error) {
   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      return nil, errors.WithStack(err);
   }

   var driver Driver = Driver{
      connector: connector,
      blockCipher: blockCipher,
      fat: make(map[dirent.Id]*dirent.Dirent),
      users: make(map[user.Id]*user.User),
      groups: make(map[group.Id]*group.Group),
      cache: nil,
      dirs: make(map[dirent.Id][]*dirent.Dirent),
      iv: iv,
      usersIV: nil,
      groupsIV: nil,
      fatIV: nil,
      cacheIV: nil,
   };

   driver.initIVs();

   // Need to init the IVs before creating the cache.
   cache, err := cache.NewMetadataCache(connector, blockCipher, driver.cacheIV);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }
   driver.cache = cache;

   return &driver, nil;
}
