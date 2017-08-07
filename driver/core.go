package driver;

// Core filesystem operations that do not operate on single files.

import (
   "time"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) Close() {
   this.connector.Close();
}

// Create a new filesystem.
func (this *Driver) CreateFilesystem(rootEmail string, rootPasshash string) error {
   this.connector.PrepareStorage();

   rootUser, err := user.New(user.ROOT_ID, rootPasshash, user.ROOT_NAME, rootEmail);
   if (err != nil) {
      golog.ErrorE("Could not create root user.", err);
      return err;
   }

   this.users[rootUser.Id] = rootUser;

   this.groups[group.EVERYBODY_ID] = group.New(group.EVERYBODY_ID, group.EVERYBODY_NAME, rootUser.Id);

   var permissions []group.Permission = []group.Permission{group.NewPermission(group.EVERYBODY_ID, true, true)};
   this.fat[dirent.ROOT_ID] = dirent.NewDir(dirent.ROOT_ID, rootUser.Id, dirent.ROOT_NAME,
         permissions, dirent.ROOT_ID, time.Now().Unix());

   // Force a write of the FAT, users, and groups.
   this.SyncToDisk();

   return nil;
}

func (this *Driver) SyncFromDisk() error {
   // TODO(eriq)
   return nil;
}

func (this *Driver) SyncToDisk() error {
   // TODO(eriq)
   return nil;
}

// Put this dirent in the semi-durable cache.
func (this *Driver) cacheDirent(direntInfo *dirent.Dirent) {
   // TODO(eriq)
}
