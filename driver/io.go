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

func (this *Driver) Read(user user.Id, file dirent.Id) (io.ReadCloser, error) {
   // TODO(eriq): This leaks permissions
   fileInfo, ok := this.fat[file];
   if (!ok) {
      return nil, NewIllegalOperationError("Cannot read non-existant file: " + string(file));
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

func (this *Driver) Put(
      user user.Id,
      name string, clearbytes io.Reader,
      groupPermissions []group.Permission, parentDir dirent.Id) error {
   if (name == "") {
      return NewIllegalOperationError("Cannot put a file with no name.");
   }

   if (groupPermissions == nil) {
      return errors.WithStack(NewIllegalOperationError("Put requires a non-nil group permissions. Empty is valid."));
   }

   // TODO(eriq): Technically, this could be leaking information about
   // the existance of a parent that the user does not have access to.
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

      err := this.checkUpdatePermissions(user, fileInfo);
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

func (this *Driver) fetchByName(name string, parent dirent.Id) *dirent.Dirent {
   // TODO(eriq)
   return nil;
}

func (this *Driver) List(user user.Id, dir dirent.Id) ([]*dirent.Dirent, error) {
   dirInfo, ok := this.fat[dir];
   if (!ok) {
      return nil, NewIllegalOperationError("Cannot list non-existant dir: " + string(dir));
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

func (this *Driver) Remove(dirent dirent.Id) error {
   // TODO(eriq)
   return nil;
}

func (this *Driver) Move(dirent dirent.Id, newParent dirent.Id) error {
   // TODO(eriq)
   return nil;
}

func (this *Driver) MakeDir(user user.Id, name string,
      parent dirent.Id, permissions []group.Permission) (dirent.Id, error) {
   if (name == "") {
      return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot make a dir with no name."));
   }

   if (permissions == nil) {
      return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("MakeDir requires a non-nil group permissions. Empty is valid."));
   }

   // TODO(eriq): Technically, this could be leaking information about
   // the existance of a parent that the user does not have access to.
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

func (this *Driver) GetDirent(user user.Id, id dirent.Id) (*dirent.Dirent, error) {
   // TODO(eriq): This leaks permissions
   info, ok := this.fat[id];
   if (!ok) {
      return nil, errors.WithStack(NewIllegalOperationError("Dirent does not exist: " + string(id)));
   }

   err := this.checkReadPermissions(user, info);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   return info, nil;
}
