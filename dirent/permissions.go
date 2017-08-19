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
