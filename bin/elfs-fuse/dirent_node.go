package main

// FUSE Nodes are higher-level file/dir operations.
// The same class (fuseDirent) is used for both nodes and handles.
// This file contains implementations of node methods.
// Implemented node interfaces:
//  - fs.Node
//  - fs.NodeStringLookuper

import (
    "os"
    "time"

    "bazil.org/fuse"
    "bazil.org/fuse/fs"
    "github.com/pkg/errors"
    "golang.org/x/net/context"

    "github.com/eriq-augustine/elfs/cipherio"
    "github.com/eriq-augustine/elfs/util"
)

const (
    FUSE_BLOCKSIZE = 512
)

func (this fuseDirent) Attr(ctx context.Context, attr *fuse.Attr) error {
    attr.Inode = 0;  // Dynamic.
    attr.Size = this.dirent.Size;
    attr.Blocks = util.CeilUint64(float64(this.dirent.Size) / FUSE_BLOCKSIZE);
    attr.Atime = time.Unix(this.dirent.AccessTimestamp, 0);
    attr.Mtime = time.Unix(this.dirent.ModTimestamp, 0);
    attr.Ctime = time.Unix(this.dirent.CreateTimestamp, 0);
    attr.Crtime = time.Unix(this.dirent.CreateTimestamp, 0);
    attr.Nlink = 1;
    attr.Uid = uint32(this.dirent.Owner);
    attr.Gid = 0;  // Group permissions are more of an ACL.
    // attr.Rdev
    // attr.Flags
    attr.BlockSize = cipherio.IO_BLOCK_SIZE;

    if (this.dirent.IsFile) {
        attr.Mode = 0555;
    } else {
        attr.Mode = os.ModeDir | 0555;
    }

    return nil;
}

func (this fuseDirent) Lookup(ctx context.Context, name string) (fs.Node, error) {
    if (this.dirent.IsFile) {
        return nil, fuse.ENOENT;
    }

    // Get the children for this dir.
    entries, err := this.driver.List(this.user.Id, this.dirent.Id);
    if (err != nil) {
        return nil, errors.Wrap(err, "Failed to list directory: " + string(this.dirent.Id));
    }

    for _, entry := range(entries) {
        if (entry.Name != name) {
            continue;
        }

        return fuseDirent{entry, this.driver, this.user}, nil;
    }

    return nil, fuse.ENOENT;
}
