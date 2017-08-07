package dirent;

import (
   "fmt"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

// Can the specified user write to this dirent.
func (this *Dirent) CanWrite(user user.Id, groups map[group.Id]*group.Group) bool {
   if (this.Owner == user) {
      return true;
   }

   for _, groupPermission := range(this.GroupPermissions) {
      if (!groupPermission.Write) {
         continue;
      }

      group, ok := groups[groupPermission.GroupId];
      if (!ok) {
         golog.Warn(fmt.Sprintf("Orphaned group permission found. Dirent: %s, Group: %d.",
               this.Id, groupPermission.GroupId));
         continue;
      }

      if (group.Users.Contains(user)) {
         return true;
      }
   }

   return false;
}
