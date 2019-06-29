package main

// FUSE Nodes are higher-level file/dir operations.
// The same class (fuseDirent) is used for both nodes and handles.
// This file contains implementations of node methods.
// Implemented node interfaces:
//  - fs.Node
//  - fs.NodeAccesser
//  - fs.NodeCreater
//  - fs.NodeFsyncer
//  - fs.NodeGetattrer
//  - fs.NodeMkdirer
//  - fs.NodeRemover
//  - fs.NodeRenamer
//  - fs.NodeSetattreattr
//  - fs.NodeStringLookuper

import (
    "bytes"
    "os"
    "time"

    "bazil.org/fuse"
    "bazil.org/fuse/fs"
    "github.com/pkg/errors"
    "golang.org/x/net/context"

    "github.com/eriq-augustine/elfs/cipherio"
    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/util"
)

const (
    FUSE_BLOCKSIZE = 512
    ACCESS_F_OK = 0
    ACCESS_R_OK = 4
    ACCESS_W_OK = 2
    ACCESS_X_OK = 1
)

func (this fuseDirent) Access(ctx context.Context, request *fuse.AccessRequest) error {
    // The mask will either be ACCESS_F_OK, or a mask of the other ACCESS_[RWX] bits.
    // See the access(2) man page.

    if (request.Mask == ACCESS_F_OK) {
        // Because of how the other FUSE API methods are implemented,
        // I do not know how the file could not exist.
        // However, we can just check with the driver again.
        info, _ := this.driver.GetDirent(this.user.Id, this.dirent.Id);
        if (info == nil) {
            return fuse.EPERM;
        }

        return nil;
    }

    group, ok := this.driver.GetGroups()[this.dirent.Group];
    if (!ok) {
        return errors.Errorf("Unable to find the group (%d) for dirent (%s).", int(this.dirent.Group), string(this.dirent.Id));
    }

    if (request.Mask & ACCESS_R_OK != 0) {
        if (!this.dirent.CanRead(this.user, group)) {
            return fuse.EPERM;
        }
    }

    if (request.Mask & ACCESS_W_OK != 0) {
        if (!this.dirent.CanWrite(this.user, group)) {
            return fuse.EPERM;
        }
    }

    if (request.Mask & ACCESS_X_OK != 0) {
        if (!this.dirent.CanExecute(this.user, group)) {
            return fuse.EPERM;
        }
    }

    return nil;
}

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
    attr.Gid = uint32(this.dirent.Group);
    // attr.Rdev
    // attr.Flags
    attr.BlockSize = cipherio.IO_BLOCK_SIZE;

    var mode os.FileMode = os.FileMode(this.dirent.Permissions);
    if (!this.dirent.IsFile) {
        mode |= os.ModeDir;
    }
    attr.Mode = mode;

    return nil;
}

// Create is only for files.
func (this fuseDirent) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
    // We will just ignore the flags, mode, and umask.
    // Since all our operations are complete, we will just write an empty file.

    // TODO(eriq): Obey the permissions from the request.

    var data []byte = make([]byte, 0);

    newFileId, err := this.driver.Put(this.user.Id, request.Name, bytes.NewReader(data), this.dirent.Id);
    if (err != nil) {
        return nil, nil, errors.Wrap(err, "Unable to create file: " + request.Name);
    }

    newFile, err := this.driver.GetDirent(this.user.Id, newFileId);
    if (err != nil) {
        return nil, nil, errors.Wrap(err, "Failed to fetch dirent: " + string(newFileId));
    }

    var entry fuseDirent = fuseDirent{newFile, this.driver, this.user};

    return entry, entry, nil;
}

func (this fuseDirent) Fsync(ctx context.Context, request *fuse.FsyncRequest) error {
    // We don't need to do anything here.
    // We already sync to the cache on all writes.
    return nil;
}

func (this fuseDirent) Getattr(ctx context.Context, request *fuse.GetattrRequest, response *fuse.GetattrResponse) error {
    return errors.WithStack(this.Attr(ctx, &response.Attr));
}

func (this fuseDirent) Lookup(ctx context.Context, name string) (fs.Node, error) {
    if (this.dirent.IsFile) {
        return nil, fuse.ENOENT;
    }

    child, err := this.driver.FetchChildByName(this.user.Id, this.dirent.Id, name);
    if (err != nil) {
        return nil, errors.WithStack(err);
    }

    if (child == nil) {
        return nil, fuse.ENOENT;
    }

    return fuseDirent{child, this.driver, this.user}, nil;
}

func (this fuseDirent) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
    // Ignore the mode and umask.

    // TODO(eriq): Obey the permissions from the request.

    newDirId, err := this.driver.MakeDir(this.user.Id, request.Name, this.dirent.Id);
    if (err != nil) {
        return nil, errors.Wrap(err, "Unable to create dir: " + request.Name);
    }

    newDir, err := this.driver.GetDirent(this.user.Id, newDirId);
    if (err != nil) {
        return nil, errors.Wrap(err, "Failed to fetch dirent: " + string(newDirId));
    }

    var entry fuseDirent = fuseDirent{newDir, this.driver, this.user};

    return entry, nil;
}

func (this fuseDirent) Remove(ctx context.Context, request *fuse.RemoveRequest) error {
    child, err := this.driver.FetchChildByName(this.user.Id, this.dirent.Id, request.Name);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (child == nil) {
        return fuse.ENOENT;
    }

    if (child.IsFile) {
        err = this.driver.RemoveFile(this.user.Id, child.Id);
    } else {
        err = this.driver.RemoveDir(this.user.Id, child.Id);
    }

    return errors.WithStack(err);
}

func (this fuseDirent) Rename(ctx context.Context, request *fuse.RenameRequest, newDir fs.Node) error {
    // Note that the context dirent is the current parent.

    var err error;
    var newParent fuseDirent = newDir.(fuseDirent);

    child, err := this.driver.FetchChildByName(this.user.Id, this.dirent.Id, request.OldName);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (child == nil) {
        return fuse.ENOENT;
    }

    // First check to see if we need to move the dirent (assign a new parent).
    if (child.Parent != newParent.dirent.Id) {
        err = this.driver.Move(this.user.Id, child.Id, newParent.dirent.Id);
        if (err != nil) {
            return errors.WithStack(err);
        }
    }

    // Now that the parent is settled, check to see if we need a rename.
    if (request.OldName != request.NewName) {
        err = this.driver.Rename(this.user.Id, child.Id, request.NewName);
        if (err != nil) {
            return errors.WithStack(err);
        }
    }

    return nil;
}

func (this fuseDirent) Setattr(ctx context.Context, request *fuse.SetattrRequest, response *fuse.SetattrResponse) error {
    // Start with the existing attributes.
    err := this.Attr(ctx, &response.Attr);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (request.Valid & fuse.SetattrMode != 0) {
        // Permissions.
        var perms dirent.Permissions = dirent.PermissionsFromFileMode(request.Mode);

        err = this.driver.ChangePermissions(this.user.Id, this.dirent.Id, perms);
        if (err != nil) {
            return errors.WithStack(err);
        }
    }

    if (request.Valid & fuse.SetattrUid != 0) {
        // Owner.
        err = this.driver.ChangeOwner(this.user.Id, this.dirent.Id, identity.UserId(request.Uid));
        if (err != nil) {
            return errors.WithStack(err);
        }
    }

    if (request.Valid & fuse.SetattrGid != 0) {
        // Group.
        err = this.driver.ChangeGroup(this.user.Id, this.dirent.Id, identity.GroupId(request.Gid));
        if (err != nil) {
            return errors.WithStack(err);
        }
    }

    if (request.Valid & fuse.SetattrSize != 0) {
        // Size.
        // Ignore, we only want to change size through writes.
    }

    if (request.Valid & fuse.SetattrAtime != 0) {
        // Access time.
        // Ignore, we already handle this internally.
    }

    if (request.Valid & fuse.SetattrMtime != 0) {
        // Mod time.
        // Ignore, we already handle this internally.
    }

    if (request.Valid & fuse.SetattrHandle != 0) {
        // Handle.
        // Ignore, we don't use handles.
    }

    return nil;
}
