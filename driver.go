package s3efs;

import (
   "io"
)

// All the key types.

// The unique name of the encrypted file.
type DirentId string;

type UserId int;
type GroupId int;

type GroupPermission struct {
   GroupId GroupId
   Read bool
   Write bool
}

// Anything that can be in a directory.
type Dirent struct {
   Id DirentId
   IsFile bool
   Owner UserId
   Name string
   CreateTimestamp int64
   ModTimestamp int64
   AccessTimestamp int64
   AccessCount uint
   GroupPermissions []GroupPermission
   Size uint64 // bytes
   Md5 string
   Dir DirentId
}

type Driver interface {
   // FS operations

   // Initialize a new filesystems at the specified address.
   Init(encryptionKey string, rootPassword string, address string) error;

   // Sync any caches to disk.
   Sync() error;

   // Dirent Operations

   // Get a readter that will read the file of the given name.
   // The reader will handle decryption and the resulting bytes will be cleartext.
   Read(file DirentId) (io.Reader, error);

   // Upsert a file.
   // The writer can stream in the clear bytes as they become available.
   // The writer will handle encryption and any metadata updates.
   Put(file DirentId, clearbytes io.Writer) error;

   // List a directory.
   List(dir DirentId) ([]*Dirent, error);

   // Remove a dirent.
   // If the dirent is ia directory, then it will be recursively removed.
   Remove(dirent DirentId) error;

   // Permission Operations

   // Change the owner of a dirent.
   ChangeOwner(dirent DirentId, newOnwer UserId) error;

   // Remove a group's access to a dirent.
   RemoveGroupAccess(dirent DirentId, group GroupId) error;

   // Upsert the permissions on a dirent for a group.
   // pserting permissions with no read or write is the same as removing access for the group.
   PutGroupAccess(dirent DirentId, permissions GroupPermission) error;

   // User Operations

   // Add a new user to the filesystem.
   // Returns the new user's id.
   // Root only.
   Useradd(name string, email string, passhash string) (UserId, error);

   // Remove a user from the filesystem.
   // All property owned by the user will inherited by root.
   // Root only.
   Userdel(user UserId) error;

   // Group Operations

   // Create a new group.
   // Returns the new group's id.
   Groupadd(name string, owner UserId) (int, error);

   // Remove a group.
   Groupdel(group GroupId) error;

   // Put a user in a group as a member.
   JoinGroup(user UserId, group GroupId) error;

   // Promote a member of the group to a group admin.
   PromoteUser(user UserId, group GroupId) error;

   // Demote a member of the group to a regular group member.
   DemoteUser(user UserId, group GroupId) error;
}
