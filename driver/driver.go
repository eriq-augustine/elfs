package driver;

// The driver is responsible for handleing all the filesystem operations.
// A driver will have a connector that will handle the operations to the actual backend
// (eg local filesystem or S3).

import (
   "crypto/aes"
   "crypto/cipher"
   "io"
   "time"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

type Connector interface {
   // Prepare the backend storage for initialization.
   PrepareStorage() error
   // Get a reader that transparently handles all decryption.
   GetEncryptedReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (io.ReadCloser, error)
   // Write out an encrypted file from cleartext bytes,
   // Manipulate NO metatdata.
   // Returns: (file size (cleartext), md5 hash (of cleartext as a hex string), error)
   Write(fileInfo *dirent.Dirent, blockCipher cipher.Block, clearbytes io.Reader) (uint64, string, error)
}

// TODO(eriq): Writes to FAT probably need a lock.

// TODO(eriq): Need to async operations and keep track of what files currently have read or writes.

type Driver struct {
   connector Connector
   blockCipher cipher.Block
   fat map[dirent.Id]*dirent.Dirent
   users map[user.Id]*user.User
   groups map[group.Id]*group.Group
}

func NewDriver(key []byte, connector Connector) (*Driver, error) {
   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      return nil, err;
   }

   var driver Driver = Driver{
      connector: connector,
      blockCipher: blockCipher,
   };

   return &driver, nil;

   // TODO(eriq): Read FAT from disk.
   // TODO(eriq): Load cache and write any changes to the disk FAT.
}

func (this *Driver) InitFilesystem(rootEmail string, rootPasshash string) error {
   this.connector.PrepareStorage();

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
         permissions, dirent.ROOT_ID, time.Now().Unix());

   // Force a write of the FAT, users, and groups.
   this.Sync();

   return nil;
}

func (this *Driver) Sync() error {
   // TODO(eriq)
   return nil;
}

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

func (this *Driver) ChangeOwner(dirent dirent.Id, newOnwer user.Id) error {
   return nil;
}

func (this *Driver) RemoveGroupAccess(dirent dirent.Id, group group.Id) error {
   return nil;
}

func (this *Driver) PutGroupAccess(dirent dirent.Id, permissions group.Permission) error {
   return nil;
}

func (this *Driver) Useradd(name string, email string, passhash string) (user.Id, error) {
   return -1, nil;
}

func (this *Driver) Userdel(user user.Id) error {
   return nil;
}

func (this *Driver) Groupadd(name string, owner user.Id) (int, error) {
   return -1, nil;
}

func (this *Driver) Groupdel(group group.Id) error {
   return nil;
}

func (this *Driver) JoinGroup(user user.Id, group group.Id) error {
   return nil;
}

func (this *Driver) PromoteUser(user user.Id, group group.Id) error {
   return nil;
}

func (this *Driver) DemoteUser(user user.Id, group group.Id) error {
   return nil;
}

// Put this dirent in the semi-durable cache.
func (this *Driver) cacheDirent(direntInfo *dirent.Dirent) {
   // TODO(eriq)
}
