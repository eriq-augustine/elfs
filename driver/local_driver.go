package driver;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "os"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

// TODO(eriq): Writes to FAT probably need a lock.

type LocalDriver struct {
   key []byte
   path string
   fat map[dirent.Id]*dirent.Dirent
   users map[user.Id]*user.User
   groups map[group.Id]*group.Group
}

// TODO(eriq): This should be returning a Driver (once we implemented all the methods).
func NewLocalDriver(key []byte, path string) (*LocalDriver, error) {
   var driver LocalDriver = LocalDriver{
      key: key,
      path: path,
   };

   return &driver, nil;
}

func (this *LocalDriver) Init(rootEmail string, rootPasshash string) error {
   os.MkdirAll(this.path, 0700);

   this.users = make(map[user.Id]*user.User);
   this.groups = make(map[group.Id]*group.Group);
   this.fat = make(map[dirent.Id]*dirent.Dirent);

   rootUser, err := user.New(user.ROOT_ID, rootPasshash, user.ROOT_NAME, rootEmail);
   if (err != nil) {
      golog.ErrorE("Could not create root user.", err);
      return err;
   }

   this.users[rootUser.Id] = rootUser;

   this.groups[group.EVERYBODY_ID] = group.New(group.EVERYBODY_ID, group.EVERYBODY_NAME, rootUser.Id);

   var permissions []group.Permission = []group.Permission{group.NewPermission(group.EVERYBODY_ID, true, true)};
   this.fat[dirent.ROOT_ID] = dirent.NewDir(dirent.ROOT_ID, rootUser.Id, dirent.ROOT_NAME,
         permissions, dirent.ROOT_ID);

   // Force a write of the FAT, users, and groups.
   this.Sync();

   return nil;
}

func (this *LocalDriver) Sync() error {
   // TODO(eriq)
   return nil;
}
