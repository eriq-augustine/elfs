package dirent;

// All the key types.

import (
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
   "github.com/eriq-augustine/s3efs/util"
)

const (
   ID_LENGTH = 32
   IV_LENGTH = 12  // Standard

   EMPTY_ID = Id("")

   ROOT_ID = EMPTY_ID
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
   Parent Id
}

func NewDir(id Id, owner user.Id, name string,
      groupPermissions []group.Permission, parent Id, timestamp int64) *Dirent {
   return &Dirent{
      Id: id,
      IsFile: false,
      IV: nil,
      Owner: owner,
      Name: name,
      CreateTimestamp: timestamp,
      ModTimestamp: timestamp,
      AccessTimestamp: timestamp,
      AccessCount: 0,
      GroupPermissions: make([]group.Permission, 0),
      Size: 0,
      Md5: "",
      Parent: parent,
   };
}

func NewFile(id Id, owner user.Id, name string,
      groupPermissions []group.Permission, parent Id, timestamp int64) *Dirent {
   return &Dirent{
      Id: id,
      IsFile: true,
      IV: util.GenIV(),
      Owner: owner,
      Name: name,
      CreateTimestamp: timestamp,
      ModTimestamp: timestamp,
      AccessTimestamp: timestamp,
      AccessCount: 0,
      GroupPermissions: groupPermissions,
      Size: 0,
      Md5: "",
      Parent: parent,
   };
}

func NewId() Id {
   return Id(util.RandomString(ID_LENGTH));
}
