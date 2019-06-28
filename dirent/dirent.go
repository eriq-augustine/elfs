package dirent;

import (
    "strings"

    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/util"
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
    Owner identity.UserId
    Group identity.GroupId
    Name string
    CreateTimestamp int64
    ModTimestamp int64
    AccessTimestamp int64
    AccessCount uint
    Permissions Permissions
    Size uint64  // bytes
    Md5 string
    Parent Id
}

func NewDir(id Id, name string, parent Id,
        owner identity.UserId, group identity.GroupId,
        timestamp int64) *Dirent {
    return &Dirent{
        Id: id,
        IsFile: false,
        IV: nil,
        Owner: owner,
        Group: group,
        Name: cleanName(name),
        CreateTimestamp: timestamp,
        ModTimestamp: timestamp,
        AccessTimestamp: timestamp,
        AccessCount: 0,
        Permissions: DEFAULT_DIR_PERMISSIONS,
        Size: 0,
        Md5: "",
        Parent: parent,
    };
}

func NewFile(id Id, name string, parent Id,
        owner identity.UserId, group identity.GroupId,
        timestamp int64) *Dirent {
    return &Dirent{
        Id: id,
        IsFile: true,
        IV: util.GenIV(),
        Owner: owner,
        Group: group,
        Name: cleanName(name),
        CreateTimestamp: timestamp,
        ModTimestamp: timestamp,
        AccessTimestamp: timestamp,
        AccessCount: 0,
        Permissions: DEFAULT_FILE_PERMISSIONS,
        Size: 0,
        Md5: "",
        Parent: parent,
    };
}

func NewId() Id {
    return Id(util.RandomString(ID_LENGTH));
}

// The only current restrictions on names is that they do not contain a newline.
func cleanName(name string) string {
    if (strings.Contains(name, "\n")) {
        return strings.Replace(name, "\n", " ", -1);
    }
    return name;
}
