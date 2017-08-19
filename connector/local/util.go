package local;

// Simple utilties.

import (
   "path"

   "github.com/eriq-augustine/golog"

   "github.com/eriq-augustine/elfs/dirent"
)

func (this *LocalConnector) getDiskPath(direntInfo *dirent.Dirent) string {
   if (direntInfo == nil) {
      golog.Panic("Cannot get path for nil dirent.");
   }

   return path.Join(this.path, string(direntInfo.Id));
}

func (this *LocalConnector) getMetadataPath(metadataId string) string {
   if (metadataId == "") {
      golog.Panic("Cannot get path for empty metadata.");
   }

   return path.Join(this.path, metadataId);
}
