package driver;

// All the key types.

import (
   "io"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

// The unique name of the encrypted file.
type Id string;

// Anything that can be in a directory.
type Dirent struct {
   Id dirent.Id
   IsFile bool
   Owner user.Id
   Name string
   CreateTimestamp int64
   ModTimestamp int64
   AccessTimestamp int64
   AccessCount uint
   GroupPermissions []group.Permission
   Size uint64 // bytes
   Md5 string
   Dir dirent.Id
}

type Driver interface {
   // FS operations

   // Initialize a new filesystems at the specified address.
   Init(rootPassword string) error;

   // Sync any caches to disk.
   Sync() error;

   // Dirent Operations

   // Get a readter that will read the file of the given name.
   // The reader will handle decryption and the resulting bytes will be cleartext.
   Read(file dirent.Id) (io.Reader, error);

   // Upsert a file.
   // The writer can stream in the clear bytes as they become available.
   // The writer will handle encryption and any metadata updates.
   Put(file dirent.Id, clearbytes io.Writer) error;

   // List a directory.
   List(dir dirent.Id) ([]*Dirent, error);

   // Remove a dirent.
   // If the dirent is ia directory, then it will be recursively removed.
   Remove(dirent dirent.Id) error;

   // Permission Operations

   // Change the owner of a dirent.
   // Root only.
   ChangeOwner(dirent dirent.Id, newOnwer user.Id) error;

   // Remove a group's access to a dirent.
   // Owner and root only.
   RemoveGroupAccess(dirent dirent.Id, group group.Id) error;

   // Upsert the permissions on a dirent for a group.
   // pserting permissions with no read or write is the same as removing access for the group.
   // Onwer and root only.
   PutGroupAccess(dirent dirent.Id, permissions group.Permission) error;

   // User Operations

   // Add a new user to the filesystem.
   // Returns the new user's id.
   // Root only.
   Useradd(name string, email string, passhash string) (user.Id, error);

   // Remove a user from the filesystem.
   // All property owned by the user will inherited by root.
   // Root only.
   Userdel(user user.Id) error;

   // Group Operations

   // Create a new group.
   // Returns the new group's id.
   Groupadd(name string, owner user.Id) (int, error);

   // Remove a group.
   // Root only.
   Groupdel(group group.Id) error;

   // Put a user in a group as a member.
   // Group admin and root only.
   JoinGroup(user user.Id, group group.Id) error;

   // Promote a member of the group to a group admin.
   // Group admin and root only.
   PromoteUser(user user.Id, group group.Id) error;

   // Demote a member of the group to a regular group member.
   // Root only.
   DemoteUser(user user.Id, group group.Id) error;
}
