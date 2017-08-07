package local;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "crypto/aes"
   "crypto/cipher"
   "io"
   "os"
   "time"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/driver"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

const (
   // When doing reads or writes, the size of data to work with in bytes.
   IO_BLOCK_SIZE = 1024 * 1024 * 4
)

// TODO(eriq): Writes to FAT probably need a lock.

// TODO(eriq): Need to async operations and keep track of what files currently have read or writes.

type LocalDriver struct {
   blockCipher cipher.Block
   path string
   fat map[dirent.Id]*dirent.Dirent
   users map[user.Id]*user.User
   groups map[group.Id]*group.Group
}

// TODO(eriq): This should be returning a Driver (once we implemented all the methods).
func NewLocalDriver(key []byte, path string) (*LocalDriver, error) {
   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      return nil, err;
   }

   var driver LocalDriver = LocalDriver{
      blockCipher: blockCipher,
      path: path,
   };

   return &driver, nil;

   // TODO(eriq): Read FAT from disk.
   // TODO(eriq): Load cache and write any changes to the disk FAT.
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
         permissions, dirent.ROOT_ID, time.Now().Unix());

   // Force a write of the FAT, users, and groups.
   this.Sync();

   return nil;
}

func (this *LocalDriver) Sync() error {
   // TODO(eriq)
   return nil;
}

func (this *LocalDriver) Read(user user.Id, file dirent.Id) (io.ReadCloser, error) {
   fileInfo, ok := this.fat[file];
   if (!ok) {
      return nil, driver.NewIllegalOperationError("Cannot read non-existant file: " + string(file));
   }

   err := this.checkReadPermissions(user, fileInfo);
   if (err != nil) {
      return nil, err;
   }

   if (!fileInfo.IsFile) {
      return nil, driver.NewIllegalOperationError("Cannot read a dir, use List() instead.");
   }

   return NewEncryptedFileReader(this.blockCipher, this.getDiskPath(fileInfo.Id), fileInfo.IV);
}

func (this *LocalDriver) Put(
      user user.Id,
      name string, clearbytes io.Reader,
      groupPermissions []group.Permission, parentDir dirent.Id) error {
   if (name == "") {
      return driver.NewIllegalOperationError("Cannot put a file with no name.");
   }

   if (groupPermissions == nil) {
      return driver.NewIllegalOperationError("Put requires a non-nil group permissions. Empty is valid.");
   }

   _, ok := this.fat[parentDir];
   if (!ok) {
      return driver.NewIllegalOperationError("Put requires an existing parent directory.");
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
         return driver.NewIllegalOperationError("Put cannot change a file's directory, use Move() instead.");
      }
   }

   fileSize, md5String, err := this.write(clearbytes, fileInfo.IV, this.getDiskPath(fileInfo.Id));
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

func (this *LocalDriver) FetchByName(name string, parent dirent.Id) *dirent.Dirent {
   return nil;
}

func (this *LocalDriver) List(dir dirent.Id) ([]*dirent.Dirent, error) {
   return nil, nil;
}

func (this *LocalDriver) Remove(dirent dirent.Id) error {
   return nil;
}

func (this *LocalDriver) Move(dirent dirent.Id, newParent dirent.Id) error {
   return nil;
}

func (this *LocalDriver) ChangeOwner(dirent dirent.Id, newOnwer user.Id) error {
   return nil;
}

func (this *LocalDriver) RemoveGroupAccess(dirent dirent.Id, group group.Id) error {
   return nil;
}

func (this *LocalDriver) PutGroupAccess(dirent dirent.Id, permissions group.Permission) error {
   return nil;
}

func (this *LocalDriver) Useradd(name string, email string, passhash string) (user.Id, error) {
   return -1, nil;
}

func (this *LocalDriver) Userdel(user user.Id) error {
   return nil;
}

func (this *LocalDriver) Groupadd(name string, owner user.Id) (int, error) {
   return -1, nil;
}

func (this *LocalDriver) Groupdel(group group.Id) error {
   return nil;
}

func (this *LocalDriver) JoinGroup(user user.Id, group group.Id) error {
   return nil;
}

func (this *LocalDriver) PromoteUser(user user.Id, group group.Id) error {
   return nil;
}

func (this *LocalDriver) DemoteUser(user user.Id, group group.Id) error {
   return nil;
}

// Put this dirent in the semi-durable cache.
func (this *LocalDriver) cacheDirent(direntInfo *dirent.Dirent) {
   // TODO(eriq)
}
