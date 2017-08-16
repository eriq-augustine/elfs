package driver;

// Operations (and helpers) that deal with file permissions.

import (
   "fmt"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) ChangeOwner(contextUser user.Id, dirent dirent.Id, newOwner user.Id) error {
   if (contextUser != user.ROOT_ID) {
      return errors.WithStack(NewIllegalOperationError("Only root can change owners."));
   }

   direntInfo, ok := this.fat[dirent];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot change the owner of a non-existant dirent."));
   }

   _, ok = this.users[newOwner];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot change owner to a non-existant user."));
   }

   if (newOwner == direntInfo.Owner) {
      return nil;
   }

   direntInfo.Owner = newOwner;
   this.cache.CacheDirentPut(direntInfo);

   return nil;
}

func (this *Driver) RemoveGroupAccess(contextUser user.Id, dirent dirent.Id, group group.Id) error {
   direntInfo, ok := this.fat[dirent];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot remove group access on a non-existant dirent."));
   }

   _, ok = this.groups[group];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot remove group access for a non-existant group."));
   }

   err := this.checkOwnerPermissions(contextUser, direntInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   _, ok = direntInfo.GroupPermissions[group];
   if (!ok) {
      return nil;
   }

   delete(direntInfo.GroupPermissions, group);
   this.cache.CacheDirentPut(direntInfo);

   return nil;
}

func (this *Driver) PutGroupAccess(contextUser user.Id, dirent dirent.Id, group group.Id, permissions group.Permission) error {
   direntInfo, ok := this.fat[dirent];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot put group access on a non-existant dirent."));
   }

   _, ok = this.groups[group];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot put group access for a non-existant group."));
   }

   err := this.checkOwnerPermissions(contextUser, direntInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   direntInfo.GroupPermissions[group] = permissions;
   this.cache.CacheDirentPut(direntInfo);

   return nil;
}

// The actual owner and root get granted permission for this.
func (this *Driver) checkOwnerPermissions(userId user.Id, direntInfo *dirent.Dirent) error {
   if (userId == user.ROOT_ID) {
      return nil;
   }

   if (userId == direntInfo.Owner) {
      return nil;
   }

   return NewPermissionsError("Need owner access.");
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

// Can the given user adminsitrate this group?
func (this *Driver) checkGroupAdminPermissions(userId user.Id, group *group.Group) error {
   if (userId == user.ROOT_ID) {
      return nil;
   }

   if (group.Admins[userId]) {
      return nil;
   }

   return NewPermissionsError(fmt.Sprintf("User (%d) cannot administrate group (%d)", int(userId), int(group.Id)));
}
