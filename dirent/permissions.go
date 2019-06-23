package dirent;

import (
   "fmt"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/elfs/group"
   "github.com/eriq-augustine/elfs/user"
)

// Can the specified user write to this dirent.
func (this *Dirent) CanWrite(userId user.Id, groups map[group.Id]*group.Group) bool {
   if (userId == user.ROOT_ID || userId == this.Owner) {
      return true;
   }

   for groupId, groupPermission := range(this.GroupPermissions) {
      if (!groupPermission.Write) {
         continue;
      }

      group, ok := groups[groupId];
      if (!ok) {
         golog.Warn(fmt.Sprintf("Orphaned group permission found. Dirent: %s, Group: %d.",
               this.Id, groupId));
         continue;
      }

      if (group.Users[userId]) {
         return true;
      }
   }

   return false;
}

// Can the specified user read the dirent.
func (this *Dirent) CanRead(userId user.Id, groups map[group.Id]*group.Group) bool {
   if (userId == user.ROOT_ID || userId == this.Owner) {
      return true;
   }

   for groupId, groupPermission := range(this.GroupPermissions) {
      if (!groupPermission.Read) {
         continue;
      }

      group, ok := groups[groupId];
      if (!ok) {
         golog.Warn(fmt.Sprintf("Orphaned group permission found. Dirent: %s, Group: %d.",
               this.Id, groupId));
         continue;
      }

      if (group.Users[userId]) {
         return true;
      }
   }

   return false;
}

// Get permissions as a standard UNIX bitset.
// For files, execute is disallowed across the board.
// For directories, execute is allowed of read is allowed.
// There is no concept of "other" in ELFS.
func (this *Dirent) UnixPermissions() uint32 {
    var permissions uint32 = 0000;

    // Owner can alwasy RW.
    permissions |= 0600;
    if (!this.IsFile) {
        permissions |= 0100;
    }

    // TODO(eriq): For group permissions, we could do an intersection of the permissions.
    // Or maybe compute based off a context user's groups.

    return permissions;
}
