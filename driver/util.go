package driver;

// Simple utilties.

import (
    "fmt"
    "math/rand"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/identity"
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

func (this *Driver) getNewUserId() identity.UserId {
    var id identity.UserId = identity.UserId(rand.Int());

    for {
        _, ok := this.users[id];
        if (!ok) {
            break;
        }

        id = identity.UserId(rand.Int());
    }

    return id;
}

func (this *Driver) getNewGroupId() identity.GroupId {
    var id identity.GroupId = identity.GroupId(rand.Int());

    for {
        _, ok := this.groups[id];
        if (!ok) {
            break;
        }

        id = identity.GroupId(rand.Int());
    }

    return id;
}

// Recursivley remove all dirents.
// Go depth first (while hitting all files along the way).
// Does not perform any permission checks.
func (this *Driver) removeDir(dir *dirent.Dirent) error {
    // First remove all children (recursively).
    for _, child := range(this.dirs[dir.Id]) {
        if (child.IsFile) {
            err := this.removeFile(child);
            if (err != nil) {
                return errors.Wrap(err, string(dir.Id));
            }
        } else {
            err := this.removeDir(child);
            if (err != nil) {
                return errors.Wrap(err, string(dir.Id));
            }
        }
    }

    // Remove from fat.
    delete(this.fat, dir.Id);

    this.cache.CacheDirentDelete(dir);

    // Remove from the dir structure (as a child).
    dirent.RemoveChild(this.dirs, dir);

    // Remove the entry from dirs (as a parent).
    delete(this.dirs, dir.Id);

    return nil;
}

// Does not perform any permission checks.
func (this *Driver) removeFile(file *dirent.Dirent) error {
    // Remove from fat first, just incase disk remove fails.
    delete(this.fat, file.Id);

    this.cache.CacheDirentDelete(file);

    // Remove from the dir structure.
    dirent.RemoveChild(this.dirs, file);

    return errors.Wrap(this.connector.RemoveFile(file), string(file.Id));
}

func (this *Driver) checkRecusiveWritePermissions(user *identity.User, group *identity.Group, direntInfo *dirent.Dirent) error {
    if (!direntInfo.CanWrite(user, group)) {
        return NewPermissionsError(fmt.Sprintf("User (%s) cannot write dirent (%s).", string(user.Id), string(direntInfo.Id)));
    }

    if (!direntInfo.IsFile) {
        for _, child := range(this.dirs[direntInfo.Id]) {
            err := this.checkRecusiveWritePermissions(user, group, child);
            if (err != nil) {
                return errors.Wrap(err, string(direntInfo.Id));
            }
        }
    }

    return nil;
}

// Get a user and dirent while performing checks for existance, permission, and type.
func (this *Driver) getUserAndDirent(
        userId identity.UserId, direntId dirent.Id,
        needRead bool, needWrite bool, needExecute bool,
        needFile bool, needDir bool) (*dirent.Dirent, *identity.User, error) {
    direntInfo, ok := this.fat[direntId];
    if (!ok) {
        return nil, nil, errors.WithStack(NewDoesntExistError(string(direntId)));
    }

    user, ok := this.users[userId];
    if (!ok) {
        return nil, nil, errors.WithStack(NewDoesntExistError(string(userId)));
    }

    direntGroup, ok := this.groups[direntInfo.Group];
    if (!ok) {
        return nil, nil, errors.Errorf("Unable to find the group (%d) for dirent (%s).", int(direntInfo.Group), string(direntId));
    }

    if (needRead && !direntInfo.CanRead(user, direntGroup)) {
        return nil, nil, NewPermissionsError(fmt.Sprintf("User (%s) cannot read dirent (%s).", string(userId), string(direntId)));
    }

    if (needWrite && !direntInfo.CanWrite(user, direntGroup)) {
        return nil, nil, NewPermissionsError(fmt.Sprintf("User (%s) cannot write dirent (%s).", string(userId), string(direntId)));
    }

    if (needExecute && !direntInfo.CanExecute(user, direntGroup)) {
        return nil, nil, NewPermissionsError(fmt.Sprintf("User (%s) cannot execute dirent (%s).", string(userId), string(direntId)));
    }

    if (needFile && !direntInfo.IsFile) {
        return nil, nil, NewIllegalOperationError(fmt.Sprintf("Dirent (%s) is not a file.", string(direntId)));
    }

    if (needDir && direntInfo.IsFile) {
        return nil, nil, NewIllegalOperationError(fmt.Sprintf("Dirent (%s) is not a directory.", string(direntId)));
    }

    return direntInfo, user, nil;
}
