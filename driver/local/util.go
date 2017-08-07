package local;

// Simple utilties.

import (
   "path"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/s3efs/dirent"
)

func (this *LocalConnector) getDiskPath(dirent *dirent.Dirent) string {
   if (dirent == nil) {
      golog.Panic("Cannot get path for nil dirent.");
   }

   return path.Join(this.path, dirent.Name);
}
