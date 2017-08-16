package driver;

// IO operations that specificially deal with single files.

import (
   "io"
   "time"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/connector"
   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) GetDirent(user user.Id, id dirent.Id) (*dirent.Dirent, error) {
   info, ok := this.fat[id];
   if (!ok) {
      return nil, errors.WithStack(NewDoesntExistError(string(id)));
   }

   err := this.checkReadPermissions(user, info);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   return info, nil;
}

func (this *Driver) List(user user.Id, dir dirent.Id) ([]*dirent.Dirent, error) {
   dirInfo, ok := this.fat[dir];
   if (!ok) {
      return nil, NewDoesntExistError(string(dir));
   }

   err := this.checkReadPermissions(user, dirInfo);
   if (err != nil) {
      return nil, err;
   }

   if (dirInfo.IsFile) {
      return nil, NewIllegalOperationError("Cannot list a file, use Read() instead.");
   }

   // Update metadata.
   dirInfo.AccessTimestamp = time.Now().Unix();
   dirInfo.AccessCount++;
   this.cache.CacheDirentPut(dirInfo);

   return this.dirs[dir], nil;
}

func (this *Driver) MakeDir(user user.Id, name string,
      parent dirent.Id, permissions map[group.Id]group.Permission) (dirent.Id, error) {
   if (name == "") {
      return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot make a dir with no name."));
   }

   if (permissions == nil) {
      return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("MakeDir requires a non-nil group permissions. Empty is valid."));
   }

   _, ok := this.fat[parent];
   if (!ok) {
      return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("MakeDir requires an existing parent directory."));
   }

   err := this.checkCreatePermissions(user, parent);
   if (err != nil) {
      return dirent.EMPTY_ID, errors.WithStack(err);
   }

   // Make sure this directory does not already exist.
   for _, child := range(this.dirs[parent]) {
      if (child.Name == name) {
         return child.Id, errors.WithStack(NewIllegalOperationError("Directory already exists: " + name));
      }
   }

   var newDir *dirent.Dirent = dirent.NewDir(this.getNewDirentId(), user, name, permissions, parent, time.Now().Unix());
   this.fat[newDir.Id] = newDir;
   this.dirs[parent] = append(this.dirs[parent], newDir);

   this.cache.CacheDirentPut(newDir);

   return newDir.Id, nil;
}

func (this *Driver) Move(user user.Id, target dirent.Id, newParent dirent.Id) error {
   // We need write permissions on the dirent and parent dir.
   targetInfo, ok := this.fat[target];
   if (!ok) {
      return errors.WithStack(NewDoesntExistError(string(target)));
   }

   newParentInfo, ok := this.fat[newParent];
   if (!ok) {
      return errors.WithStack(NewDoesntExistError(string(newParent)));
   }

   err := this.checkWritePermissions(user, targetInfo);
   if (err != nil) {
      return errors.Wrap(err, string(target));
   }

   err = this.checkWritePermissions(user, newParentInfo);
   if (err != nil) {
      return errors.Wrap(err, string(newParent));
   }

   if (newParentInfo.IsFile) {
      return errors.WithStack(NewIllegalOperationError("Cannot move a dirent into a file, need a dir."));
   }

   if (targetInfo.Parent == newParent) {
      return nil;
   }

   // Update dir structure: remove old reference, add new one.
   dirent.RemoveChild(this.dirs, targetInfo);
   this.dirs[newParent] = append(this.dirs[newParent], targetInfo);

   // Update fat
   targetInfo.Parent = newParent;
   this.cache.CacheDirentPut(targetInfo);

   return nil;
}

func (this *Driver) Put(
      user user.Id,
      name string, clearbytes io.Reader,
      groupPermissions map[group.Id]group.Permission, parentDir dirent.Id) error {
   if (name == "") {
      return NewIllegalOperationError("Cannot put a file with no name.");
   }

   if (groupPermissions == nil) {
      return errors.WithStack(NewIllegalOperationError("Put requires a non-nil group permissions. Empty is valid."));
   }

   _, ok := this.fat[parentDir];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Put requires an existing parent directory."));
   }

   // Consider all parts of this operation happening at this timestamp.
   var operationTimestamp int64 = time.Now().Unix();

   var fileInfo *dirent.Dirent = this.fetchByName(name, parentDir);
   var newFile bool;

   // Create or update?
   if (fileInfo == nil) {
      // Create
      newFile = true;

      err := this.checkCreatePermissions(user, parentDir);
      if (err != nil) {
         return errors.WithStack(err);
      }

      fileInfo = dirent.NewFile(this.getNewDirentId(), user, name, groupPermissions, parentDir, operationTimestamp);
   } else {
      // Update
      newFile = false;

      err := this.checkWritePermissions(user, fileInfo);
      if (err != nil) {
         return errors.WithStack(err);
      }

      if (!fileInfo.IsFile) {
         return errors.WithStack(NewIllegalOperationError("Put cannot write a directory, do you mean to MkDir()?"));
      }

      if (parentDir != fileInfo.Parent) {
         return NewIllegalOperationError("Put cannot change a file's directory, use Move() instead.");
      }
   }

   fileSize, md5String, err := connector.Write(this.connector, fileInfo, this.blockCipher, clearbytes);
   if (err != nil) {
      return err;
   }

   // Update metadata.
   // Note that some of the data is available before the write,
   // but we only want to update the metatdata if the write goes through.
   fileInfo.ModTimestamp = operationTimestamp;
   fileInfo.AccessTimestamp = operationTimestamp;
   fileInfo.AccessCount++;
   fileInfo.Size = fileSize;
   fileInfo.Md5 = md5String;
   fileInfo.Parent = parentDir;
   fileInfo.GroupPermissions = groupPermissions;

   // If this file is new, we need to make sure it is in that memory-FAT.
   this.fat[fileInfo.Id] = fileInfo;

   // Update the directory tree if this is a new file.
   if (newFile) {
      this.dirs[parentDir] = append(this.dirs[parentDir], fileInfo);
   }

   this.cache.CacheDirentPut(fileInfo);

   return nil;
}

func (this *Driver) Read(user user.Id, file dirent.Id) (io.ReadCloser, error) {
   fileInfo, ok := this.fat[file];
   if (!ok) {
      return nil, NewDoesntExistError(string(file));
   }

   err := this.checkReadPermissions(user, fileInfo);
   if (err != nil) {
      return nil, err;
   }

   if (!fileInfo.IsFile) {
      return nil, NewIllegalOperationError("Cannot read a dir, use List() instead.");
   }

   reader, err := this.connector.GetCipherReader(fileInfo, this.blockCipher);
   if (err != nil) {
      return nil, err;
   }

   // Update metadata.
   fileInfo.AccessTimestamp = time.Now().Unix();
   fileInfo.AccessCount++;
   this.cache.CacheDirentPut(fileInfo);

   return reader, nil;
}

func (this *Driver) RemoveDir(user user.Id, dir dirent.Id) error {
   dirInfo, ok := this.fat[dir];
   if (!ok) {
      return NewDoesntExistError(string(dir));
   }

   err := this.checkRecusiveWritePermissions(user, dirInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   if (dirInfo.IsFile) {
      return NewIllegalOperationError("Cannot use RemoveDir() for a file, use RemoveFile() instead.");
   }

   return errors.WithStack(this.removeDir(dirInfo));
}

func (this *Driver) RemoveFile(user user.Id, file dirent.Id) error {
   fileInfo, ok := this.fat[file];
   if (!ok) {
      return NewDoesntExistError(string(file));
   }

   err := this.checkWritePermissions(user, fileInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   if (!fileInfo.IsFile) {
      return NewIllegalOperationError("Cannot use RemoveFile() for a dir, use RemoveDir() instead.");
   }

   return errors.WithStack(this.removeFile(fileInfo));
}

func (this *Driver) Rename(user user.Id, target dirent.Id, newName string) error {
   if (newName == "") {
      return errors.WithStack(NewIllegalOperationError("Cannot put a file with no name."));
   }

   targetInfo, ok := this.fat[target];
   if (!ok) {
      return errors.WithStack(NewDoesntExistError(string(target)));
   }

   err := this.checkWritePermissions(user, targetInfo);
   if (err != nil) {
      return errors.Wrap(err, string(target));
   }

   if (newName == targetInfo.Name) {
      return nil;
   }

   // Update fat
   targetInfo.Name = newName;
   this.cache.CacheDirentPut(targetInfo);

   return nil;
}

func (this *Driver) fetchByName(name string, parent dirent.Id) *dirent.Dirent {
   for _, child := range(this.dirs[parent]) {
      if (child.Name == name) {
         return child;
      }
   }

   return nil;
}

// Recursivley remove all dirents.
// Go depth first (while hitting all files along the way).
// Does not perform any permission checks.
func (this *Driver) removeDir(dir *dirent.Dirent) error {
   // First remove all children (recursively).
   for _, child := range(this.dirs[dir.Id]) {
      if (child.IsFile) {
         err := this.removeFile(child);
         if (err != nil) {
            return errors.Wrap(err, string(dir.Id));
         }
      } else {
         err := this.removeDir(child);
         if (err != nil) {
            return errors.Wrap(err, string(dir.Id));
         }
      }
   }

   // Remove from fat.
   delete(this.fat, dir.Id);

   this.cache.CacheDirentDelete(dir);

   // Remove from the dir structure (as a child).
   dirent.RemoveChild(this.dirs, dir);

   // Remove the entry from dirs (as a parent).
   delete(this.dirs, dir.Id);

   return nil;
}

// Does not perform any permission checks.
func (this *Driver) removeFile(file *dirent.Dirent) error {
   // Remove from fat first, just incase disk remove fails.
   delete(this.fat, file.Id);

   this.cache.CacheDirentDelete(file);

   // Remove from the dir structure.
   dirent.RemoveChild(this.dirs, file);

   return errors.Wrap(this.connector.RemoveFile(file), string(file.Id));
}
