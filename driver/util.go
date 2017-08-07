package driver;

// Simple utilties.

import (
   "github.com/eriq-augustine/s3efs/dirent"
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
