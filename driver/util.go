package driver;

// Simple utilties.

import (
   "math/rand"

   "github.com/eriq-augustine/elfs/dirent"
   "github.com/eriq-augustine/elfs/group"
   "github.com/eriq-augustine/elfs/user"
)

// Get a new, available dirent id.
func (this *Driver) getNewDirentId() dirent.Id {
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

func (this *Driver) getNewUserId() user.Id {
   var id user.Id = user.Id(rand.Int());

   for {
      _, ok := this.users[id];
      if (!ok) {
         break;
      }

      id = user.Id(rand.Int());
   }

   return id;
}

func (this *Driver) getNewGroupId() group.Id {
   var id group.Id = group.Id(rand.Int());

   for {
      _, ok := this.groups[id];
      if (!ok) {
         break;
      }

      id = group.Id(rand.Int());
   }

   return id;
}
