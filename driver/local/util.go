package local;

// Simple utilties.

import (
   "path"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
)

// Get a new, available dirent id.
func (this *LocalDriver) getNewDirentId() dirent.Id {
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

func (this *LocalDriver) getDiskPath(dirent dirent.Id) string {
   info, ok := this.fat[dirent];
   if (!ok) {
      golog.Panic("Cannot get path for non-existant dirent.");
   }

   return path.Join(this.path, info.Name);
}
