package dirent;

// All the key types.

import (
   "time"

   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
   "github.com/eriq-augustine/s3efs/util"
)

const (
   ID_LENGTH = 32
   IV_LENGTH = 12  // Standard

   ROOT_ID = Id(0)
   ROOT_NAME = ""
)

// The unique name of the encrypted file.
type Id string;

// Anything that can be in a directory.
type Dirent struct {
   Id Id
   IsFile bool
   IV []byte
   Owner user.Id
   Name string
   CreateTimestamp int64
   ModTimestamp int64
   AccessTimestamp int64
   AccessCount uint
   GroupPermissions []group.Permission
   Size uint64 // bytes
   Md5 string
   Dir Id
}

func NewDir(id Id, owner user.Id, name string, groupPermissions []group.Permission, parent Id) *Dirent {
   return &Dirent{
      Id: id,
      IsFile: false,
      IV: util.GenIV(),
      Owner: owner,
      Name: name,
      CreateTimestamp: time.Now().Unix(),
      ModTimestamp: time.Now().Unix(),
      AccessTimestamp: time.Now().Unix(),
      AccessCount: 0,
      GroupPermissions: make([]group.Permission, 0),
      Size: 0,
      Md5: "",
      Dir: parent,
   };
}

func NewId() Id {
   return Id(util.RandomString(ID_LENGTH));
}
