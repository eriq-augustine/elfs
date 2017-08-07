package driver;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "crypto/aes"
   "crypto/cipher"
   "crypto/md5"
   "fmt"
   "hash"
   "io"
   "os"
   "path"
   "time"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
   "github.com/eriq-augustine/s3efs/util"
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
      return nil, NewIllegalOperationError("Cannot read non-existant file: " + string(file));
   }

   err := this.checkReadPermissions(user, fileInfo);
   if (err != nil) {
      return nil, err;
   }

   if (!fileInfo.IsFile) {
      return nil, NewIllegalOperationError("Cannot read a dir, use List() instead.");
   }

   return NewEncryptedFileReader(this.blockCipher, this.getDiskPath(fileInfo.Id), fileInfo.IV);
}

func (this *LocalDriver) Put(
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

// Helpers

// Put this dirent in the semi-durable cache.
func (this *LocalDriver) cacheDirent(direntInfo *dirent.Dirent) {
   // TODO(eriq)
}

// Get a new, available dirent id.
func (this *LocalDriver) getNewDirentId() dirent.Id {
   var id dirent.Id = dirent.NewId();

   for {
      _, ok := this.fat[id];
      if (!ok) {
         break;
      }

      id = dirent.NewId();
   }

   return id;
}

func (this *LocalDriver) getDiskPath(dirent dirent.Id) string {
   info, ok := this.fat[dirent];
   if (!ok) {
      golog.Panic("Cannot get path for non-existant dirent.");
   }

   return path.Join(this.path, info.Name);
}

// Write some general cleartext to disk.
// All the metadata management will be left out since we could be writing the
// FAT which itself does not have any metadata.
// Returns: (file size, md5 hash (hex string), error).
func (this *LocalDriver) write(clearbytes io.Reader, rawIV []byte, path string) (uint64, string, error) {
   // TODO(eriq): Do we need to create a different GCM (AEAD) every time?
   gcm, err := cipher.NewGCM(this.blockCipher);
   if err != nil {
      return 0, "", err;
   }

   fileWriter, err := os.Create(path);
   if (err != nil) {
      golog.ErrorE("Unable to create file on disk at: " + path, err);
      return 0, "", err;
   }
   defer fileWriter.Close();

   err = fileWriter.Chmod(0600);
   if (err != nil) {
      golog.ErrorE("Unable to change file permissions of: " + path, err);
      return 0, "", err;
   }

   // Make a copy of the IV since we will be incrementing it for each chunk.
   var iv []byte = append([]byte(nil), rawIV...);

   // Allocate enough room for the cleatext and ciphertext.
   var buffer []byte = make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead());
   var fileSize uint64 = 0;
   var m5dHash hash.Hash = md5.New();

   var done bool = false;
   for (!done) {
      // Always resize (not reallocate) to the block size.
      readSize, err := clearbytes.Read(buffer[0:IO_BLOCK_SIZE]);
      if (err != nil) {
         if (err == io.EOF) {
            done = true;
         } else {
            return 0, "", err;
         }
      }

      // Keep track of the size and hash.
      fileSize += uint64(readSize);
      m5dHash.Write(buffer);

      if (readSize > 0) {
         // Reuse the buffer for the cipertext.
         gcm.Seal(buffer[:0], iv, buffer[0:readSize], nil);
         _, err := fileWriter.Write(buffer);
         if (err != nil) {
            golog.ErrorE("Failed to write file block for: " + path, err);
            return 0, "", err;
         }

         // Prepare the IV for the next encrypt.
         util.IncrementBytes(iv);
      }
   }

   return fileSize, fmt.Sprintf("%x", m5dHash.Sum(nil)), nil;
}

// A Reader that will read an encrypted file, decrypt them, and return the cleartext
// all in chunks of size IO_BLOCK_SIZE.
// Note that the cleartext will be in checks of IO_BLOCK_SIZE,
// but the cipertext read will be slightly larger.
type EncryptedFileReader struct {
   gcm cipher.AEAD
   buffer []byte
   iv []byte
   fileReader *os.File
   done bool
}

func NewEncryptedFileReader(
      blockCipher cipher.Block,
      path string, rawIV []byte,
      ) (*EncryptedFileReader, error) {
   // TODO(eriq): Do we need to create a different GCM (AEAD) every time?
   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, err;
   }

   fileReader, err := os.Open(path);
   if (err != nil) {
      golog.ErrorE("Unable to open file on disk at: " + path, err);
      return nil, err;
   }

   var rtn EncryptedFileReader = EncryptedFileReader{
      gcm: gcm,
      // Allocate enough room for the ciphertext.
      buffer: make([]byte, 0, IO_BLOCK_SIZE + gcm.Overhead()),
      // Make a copy of the IV since we will be incrementing it for each chunk.
      iv: append([]byte(nil), rawIV...),
      fileReader: fileReader,
      done: false,
   };

   return &rtn, nil;
}

func (this *EncryptedFileReader) Read(outBuffer []byte) (int, error) {
   if (this.done) {
      return 0, io.EOF;
   }

   if (cap(outBuffer) < IO_BLOCK_SIZE) {
      return 0, fmt.Errorf("Buffer for EncryptedFileReader is too small. Must be at least %d.", IO_BLOCK_SIZE);
   }

   // Resize the buffer (without allocating) to ensure we only read exactly what we want.
   this.buffer = this.buffer[0:IO_BLOCK_SIZE + this.gcm.Overhead()];

   // Get the ciphertext.
   _, err := this.fileReader.Read(this.buffer);
   if (err != nil) {
      if (err != io.EOF) {
         return 0, err;
      }

      this.done = true;
   }

   // Resize the destination so we can reliably check the output size.
   outBuffer = outBuffer[0:0];

   _, err = this.gcm.Open(outBuffer, this.iv, this.buffer, nil);
   if (err != nil) {
      golog.ErrorE("Failed to decrypt file.", err);
      return 0, err;
   }

   // Prepare the IV for the next decrypt.
   util.IncrementBytes(this.iv);

   return len(outBuffer), nil;
}

func (this *EncryptedFileReader) Close() error {
   return this.fileReader.Close();
}

// Helpers specifically for permissions.

// To create a file, we only need write on the parent directory.
func (this *LocalDriver) checkCreatePermissions(user user.Id, parentDir dirent.Id) error {
   if (!this.fat[parentDir].CanWrite(user, this.groups)) {
      return NewPermissionsError("Cannot create a dirent in a directory you cannot write in.");
   }

   return nil;
}

// To update a file's contents, we need write on the file itself (but not the parent).
func (this *LocalDriver) checkUpdatePermissions(user user.Id, fileInfo *dirent.Dirent) error {
   if (!fileInfo.CanWrite(user, this.groups)) {
      return NewPermissionsError("Cannot update a file you cannot write to.");
   }

   return nil;
}

// Simple read check.
func (this *LocalDriver) checkReadPermissions(user user.Id, fileInfo *dirent.Dirent) error {
   if (!fileInfo.CanRead(user, this.groups)) {
      return NewPermissionsError("No read premissions.");
   }

   return nil;
}
