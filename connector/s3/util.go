package s3;

// Simple utilties.

import (
    "path"

    "github.com/eriq-augustine/golog"

    "github.com/eriq-augustine/elfs/connector"
    "github.com/eriq-augustine/elfs/dirent"
)

const (
    LOCK_FILENAME = "remote_lock"
)

func (this *S3Connector) getDataPath(direntInfo *dirent.Dirent) string {
    if (direntInfo == nil) {
        golog.Panic("Cannot get path for nil dirent.");
    }

    var prefix string = string(direntInfo.Id)[0:connector.DATA_GROUP_PREFIX_LEN];

    return path.Join(connector.FS_SYS_DIR_DATA, prefix, string(direntInfo.Id));
}

func (this *S3Connector) getMetadataPath(metadataId string) string {
    if (metadataId == "") {
        golog.Panic("Cannot get path for empty metadata.");
    }

    return path.Join(connector.FS_SYS_DIR_ADMIN, metadataId);
}

func (this *S3Connector) getLockPath() string {
    return path.Join(connector.FS_SYS_DIR_ADMIN, LOCK_FILENAME);
}
