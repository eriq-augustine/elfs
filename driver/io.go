package driver;

// IO operations that specificially deal with single files.

import (
   "io"
   "time"

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

   return this.connector.GetEncryptedReader(fileInfo, this.blockCipher);
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

   // Create or update?
   if (fileInfo == nil) {
      // Create
      err := this.checkCreatePermissions(user, parentDir);
      if (err != nil) {
         return err;
      }

      fileInfo = dirent.NewFile(this.getNewDirentId(), user, name, groupPermissions, parentDir, operationTimestamp);
   } else {
      // Update
      err := this.checkUpdatePermissions(user, fileInfo);
      if (err != nil) {
         return err;
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

   this.cacheDirent(fileInfo);

   return nil;
}

func (this *Driver) FetchByName(name string, parent dirent.Id) *dirent.Dirent {
   return nil;
}

func (this *Driver) List(dir dirent.Id) ([]*dirent.Dirent, error) {
   return nil, nil;
}

func (this *Driver) Remove(dirent dirent.Id) error {
   return nil;
}

func (this *Driver) Move(dirent dirent.Id, newParent dirent.Id) error {
   return nil;
}
