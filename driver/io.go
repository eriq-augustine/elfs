package driver;

// IO operations that specificially deal with single files.

import (
   "io"
   "time"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) Read(user user.Id, file dirent.Id) (io.ReadCloser, error) {
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

   reader, err := this.connector.GetEncryptedReader(fileInfo, this.blockCipher);
   if (err != nil) {
      return nil, err;
   }

   // Update metadata.
   fileInfo.AccessTimestamp = time.Now().Unix();
   fileInfo.AccessCount++;
   this.cacheDirent(fileInfo);

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
      return NewIllegalOperationError("Put requires a non-nil group permissions. Empty is valid.");
   }

   _, ok := this.fat[parentDir];
   if (!ok) {
      return NewIllegalOperationError("Put requires an existing parent directory.");
   }

   // Consider all parts of this operation happening at this timestamp.
   var operationTimestamp int64 = time.Now().Unix();

   var fileInfo *dirent.Dirent = this.FetchByName(name, parentDir);
   var newFile bool;

   // Create or update?
   if (fileInfo == nil) {
      // Create
      newFile = true;

      err := this.checkCreatePermissions(user, parentDir);
      if (err != nil) {
         return err;
      }

      fileInfo = dirent.NewFile(this.getNewDirentId(), user, name, groupPermissions, parentDir, operationTimestamp);
   } else {
      // Update
      newFile = false;

      err := this.checkUpdatePermissions(user, fileInfo);
      if (err != nil) {
         return err;
      }

      if (!fileInfo.IsFile) {
         return errors.WithStack(NewIllegalOperationError("Put cannot write a directory, do you mean to MkDir()?"));
      }

      if (parentDir != fileInfo.Parent) {
         return NewIllegalOperationError("Put cannot change a file's directory, use Move() instead.");
      }
   }

   fileSize, md5String, err := this.connector.Write(fileInfo, this.blockCipher, clearbytes);
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
      this.root.AddNode(this.fat, fileInfo);
   }

   this.cacheDirent(fileInfo);

   return nil;
}

func (this *Driver) FetchByName(name string, parent dirent.Id) *dirent.Dirent {
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

   path, err := dirent.GetPath(this.fat, dir);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to get path for " + string(dir));
   }

   node, err := this.root.GetNode(path);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to get node for " + string(dir));
   }

   var dirents []*dirent.Dirent = make([]*dirent.Dirent, 0, len(node.Children));
   for _, child := range(node.Children) {
      dirents = append(dirents, this.fat[child.Id]);
   }

   return dirents, nil;
}

func (this *Driver) Remove(dirent dirent.Id) error {
   return nil;
}

func (this *Driver) Move(dirent dirent.Id, newParent dirent.Id) error {
   return nil;
}
