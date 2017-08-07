package local;

// Helpers specifically for permissions.

import (
   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/driver"
   "github.com/eriq-augustine/s3efs/user"
)


// To create a file, we only need write on the parent directory.
func (this *LocalDriver) checkCreatePermissions(user user.Id, parentDir dirent.Id) error {
   if (!this.fat[parentDir].CanWrite(user, this.groups)) {
      return driver.NewPermissionsError("Cannot create a dirent in a directory you cannot write in.");
   }

   return nil;
}

// To update a file's contents, we need write on the file itself (but not the parent).
func (this *LocalDriver) checkUpdatePermissions(user user.Id, fileInfo *dirent.Dirent) error {
   if (!fileInfo.CanWrite(user, this.groups)) {
      return driver.NewPermissionsError("Cannot update a file you cannot write to.");
   }

   return nil;
}

// Simple read check.
func (this *LocalDriver) checkReadPermissions(user user.Id, fileInfo *dirent.Dirent) error {
   if (!fileInfo.CanRead(user, this.groups)) {
      return driver.NewPermissionsError("No read premissions.");
   }

   return nil;
}
