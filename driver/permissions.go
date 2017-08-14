package driver;

// Operations (and helpers) that deal with file permissions.

import (
   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) ChangeOwner(dirent dirent.Id, newOnwer user.Id) error {
   return nil;
}

func (this *Driver) RemoveGroupAccess(dirent dirent.Id, group group.Id) error {
   return nil;
}

func (this *Driver) PutGroupAccess(dirent dirent.Id, permissions group.Permission) error {
   return nil;
}

// To create a file, we only need write on the parent directory.
func (this *Driver) checkCreatePermissions(user user.Id, parentDir dirent.Id) error {
   if (!this.fat[parentDir].CanWrite(user, this.groups)) {
      return NewPermissionsError("Cannot create a dirent in a directory you cannot write in.");
   }

   return nil;
}

// To update a file's contents, we need write on the file itself (but not the parent).
func (this *Driver) checkWritePermissions(user user.Id, fileInfo *dirent.Dirent) error {
   if (!fileInfo.CanWrite(user, this.groups)) {
      return NewPermissionsError("Cannot update a file you cannot write to.");
   }

   return nil;
}

// Simple read check.
func (this *Driver) checkReadPermissions(user user.Id, fileInfo *dirent.Dirent) error {
   if (!fileInfo.CanRead(user, this.groups)) {
      return NewPermissionsError("No read premissions.");
   }

   return nil;
}

func (this *Driver) checkRecusiveWritePermissions(user user.Id, fileInfo *dirent.Dirent) error {
   err := this.checkWritePermissions(user, fileInfo);
   if (err != nil) {
      return errors.Wrap(err, string(fileInfo.Id));
   }

   if (!fileInfo.IsFile) {
      for _, child := range(this.dirs[fileInfo.Id]) {
         err = this.checkRecusiveWritePermissions(user, child);
         if (err != nil) {
            return errors.Wrap(err, string(fileInfo.Id));
         }
      }
   }

   return nil;
}
