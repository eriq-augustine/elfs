package driver;

// IO operations that specificially deal with single files.

import (
    "fmt"
    "io"
    "time"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/connector"
    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/util"
)

func (this *Driver) GetDirent(userId identity.UserId, direntId dirent.Id) (*dirent.Dirent, error) {
    direntInfo, _, err := this.getUserAndDirent(userId, direntId, true, false, false, false, false);
    if (err != nil) {
        return nil, errors.WithStack(err);
    }

    return direntInfo, nil;
}

func (this *Driver) List(userId identity.UserId, direntId dirent.Id) ([]*dirent.Dirent, error) {
    direntInfo, _, err := this.getUserAndDirent(userId, direntId, true, false, true, false, true);
    if (err != nil) {
        return nil, errors.WithStack(err);
    }

    // Update metadata.
    direntInfo.AccessTimestamp = time.Now().Unix();
    direntInfo.AccessCount++;
    this.cache.CacheDirentPut(direntInfo);

    return this.dirs[direntId], nil;
}

func (this *Driver) MakeDir(userId identity.UserId, name string, parentId dirent.Id) (dirent.Id, error) {
    if (name == "") {
        return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot make a dir with no name."));
    }

    _, user, err := this.getUserAndDirent(userId, parentId, false, true, false, false, true);
    if (err != nil) {
        return dirent.EMPTY_ID, errors.WithStack(err);
    }

    // Make sure this directory does not already exist.
    for _, child := range(this.dirs[parentId]) {
        if (child.Name == name) {
            return child.Id, errors.WithStack(NewIllegalOperationError("Directory already exists: " + name));
        }
    }

    var newDir *dirent.Dirent = dirent.NewDir(this.getNewDirentId(), name, parentId, userId, user.Usergroup, time.Now().Unix());
    this.fat[newDir.Id] = newDir;
    this.dirs[parentId] = append(this.dirs[parentId], newDir);

    this.cache.CacheDirentPut(newDir);

    return newDir.Id, nil;
}

func (this *Driver) Move(userId identity.UserId, targetId dirent.Id, newParentId dirent.Id) error {
    targetInfo, _, err := this.getUserAndDirent(userId, targetId, false, true, false, false, false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    _, _, err = this.getUserAndDirent(userId, newParentId, false, true, false, false, true);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (targetInfo.Parent == newParentId) {
        return nil;
    }

    // Update dir structure: remove old reference, add new one.
    dirent.RemoveChild(this.dirs, targetInfo);
    this.dirs[newParentId] = append(this.dirs[newParentId], targetInfo);

    // Update fat
    targetInfo.Parent = newParentId;
    this.cache.CacheDirentPut(targetInfo);

    return nil;
}

func (this *Driver) Put(
        userId identity.UserId,
        name string, clearbytes io.Reader,
        parentId dirent.Id) (dirent.Id, error) {
    if (name == "") {
        return dirent.EMPTY_ID, NewIllegalOperationError("Cannot put a file with no name.");
    }

    parentInfo, user, err := this.getUserAndDirent(userId, parentId, false, false, false, false, false);
    if (err != nil) {
        return dirent.EMPTY_ID, errors.WithStack(err);
    }

    // Consider all parts of this operation happening at this timestamp.
    var operationTimestamp int64 = time.Now().Unix();

    fileInfo, err := this.FetchChildByName(userId, parentId, name);
    if (err != nil) {
        return dirent.EMPTY_ID, errors.WithStack(err);
    }

    var newFile bool;
    var permissions dirent.Permissions = dirent.EMPTY_PERMISSIONS;

    // Create or update?
    if (fileInfo == nil) {
        // Create
        newFile = true;
        permissions = dirent.DEFAULT_FILE_PERMISSIONS;

        parentGroup, ok := this.groups[parentInfo.Group];
        if (!ok) {
            return dirent.EMPTY_ID, errors.Errorf("Unable to find the group (%d) for dirent (%s).", int(parentInfo.Group), string(parentId));
        }

        if (!parentInfo.CanWrite(user, parentGroup)) {
            return dirent.EMPTY_ID, NewPermissionsError(fmt.Sprintf("User (%s) cannot write to the parent (%s).", string(userId), string(parentId)));
        }

        fileInfo = dirent.NewFile(this.getNewDirentId(), name, parentId, userId, user.Usergroup, operationTimestamp);
    } else {
        // Update
        newFile = false;
        permissions = fileInfo.Permissions;

        fileGroup, ok := this.groups[fileInfo.Group];
        if (!ok) {
            return dirent.EMPTY_ID, errors.Errorf("Unable to find the group (%d) for dirent (%s).", int(fileInfo.Group), string(fileInfo.Id));
        }

        if (!fileInfo.CanWrite(user, fileGroup)) {
            return dirent.EMPTY_ID, NewPermissionsError(fmt.Sprintf("User (%s) cannot write to the parent (%s).", string(userId), string(parentId)));
        }

        if (!fileInfo.IsFile) {
            return dirent.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Put cannot write a directory, do you mean to MakeDir()?"));
        }

        if (parentId != fileInfo.Parent) {
            return dirent.EMPTY_ID, NewIllegalOperationError("Put cannot change a file's directory, use Move() instead.");
        }
    }

    fileSize, md5String, err := connector.Write(this.connector, fileInfo, this.blockCipher, clearbytes);
    if (err != nil) {
        return dirent.EMPTY_ID, err;
    }

    // Update metadata.
    // Note that some of the data is available before the write,
    // but we only want to update the metatdata if the write goes through.
    fileInfo.ModTimestamp = operationTimestamp;
    fileInfo.AccessTimestamp = operationTimestamp;
    fileInfo.AccessCount++;
    fileInfo.Size = fileSize;
    fileInfo.Md5 = md5String;
    fileInfo.Parent = parentId;
    fileInfo.Permissions = permissions;

    // If this file is new, we need to make sure it is in that memory-FAT.
    this.fat[fileInfo.Id] = fileInfo;

    // Update the directory tree if this is a new file.
    if (newFile) {
        this.dirs[parentId] = append(this.dirs[parentId], fileInfo);
    }

    this.cache.CacheDirentPut(fileInfo);

    return fileInfo.Id, nil;
}

func (this *Driver) Read(userId identity.UserId, fileId dirent.Id) (util.ReadSeekCloser, error) {
    fileInfo, _, err := this.getUserAndDirent(userId, fileId, true, false, false, true, false);
    if (err != nil) {
        return nil, errors.WithStack(err);
    }

    reader, err := this.connector.GetCipherReader(fileInfo, this.blockCipher);
    if (err != nil) {
        return nil, err;
    }

    // Update metadata.
    fileInfo.AccessTimestamp = time.Now().Unix();
    fileInfo.AccessCount++;
    this.cache.CacheDirentPut(fileInfo);

    return reader, nil;
}

func (this *Driver) RemoveDir(userId identity.UserId, dirId dirent.Id) error {
    dirInfo, user, err := this.getUserAndDirent(userId, dirId, false, true, false, false, true);
    if (err != nil) {
        return errors.WithStack(err);
    }

    group, ok := this.groups[dirInfo.Group];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Unable to find a dirent's group."));
    }

    err = this.checkRecusiveWritePermissions(user, group, dirInfo);
    if (err != nil) {
        return errors.WithStack(err);
    }

    return errors.WithStack(this.removeDir(dirInfo));
}

func (this *Driver) RemoveFile(userId identity.UserId, fileId dirent.Id) error {
    fileInfo, _, err := this.getUserAndDirent(userId, fileId, false, true, false, true, false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    return errors.WithStack(this.removeFile(fileInfo));
}

func (this *Driver) Rename(userId identity.UserId, targetId dirent.Id, newName string) error {
    if (newName == "") {
        return errors.WithStack(NewIllegalOperationError("Cannot rename to an empty name."));
    }

    targetInfo, _, err := this.getUserAndDirent(userId, targetId, false, true, false, false, false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (newName == targetInfo.Name) {
        return nil;
    }

    // Update fat
    targetInfo.Name = newName;
    this.cache.CacheDirentPut(targetInfo);

    return nil;
}

func (this *Driver) ChangeOwner(userId identity.UserId, direntId dirent.Id, newOwnerId identity.UserId) error {
    direntInfo, _, err := this.getUserAndDirent(userId, direntId, false, false, false, false, false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (userId != identity.ROOT_USER_ID || userId != direntInfo.Owner) {
        return errors.WithStack(NewIllegalOperationError("Only owner/root can change owners."));
    }

    _, ok := this.users[newOwnerId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot change owner to a non-existant user."));
    }

    if (newOwnerId == direntInfo.Owner) {
        return nil;
    }

    direntInfo.Owner = newOwnerId;
    this.cache.CacheDirentPut(direntInfo);

    return nil;
}

func (this *Driver) ChangeGroup(userId identity.UserId, direntId dirent.Id, newGroupId identity.GroupId) error {
    direntInfo, _, err := this.getUserAndDirent(userId, direntId, false, false, false, false, false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (userId != identity.ROOT_USER_ID || userId != direntInfo.Owner) {
        return errors.WithStack(NewIllegalOperationError("Only owner/root can change groups."));
    }

    _, ok := this.groups[newGroupId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot change group to a non-existant group."));
    }

    if (newGroupId == direntInfo.Group) {
        return nil;
    }

    direntInfo.Group = newGroupId;
    this.cache.CacheDirentPut(direntInfo);

    return nil;
}

func (this *Driver) ChangePermissions(userId identity.UserId, direntId dirent.Id, perms dirent.Permissions) error {
    direntInfo, _, err := this.getUserAndDirent(userId, direntId, false, false, false, false, false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (userId != identity.ROOT_USER_ID || userId != direntInfo.Owner) {
        return errors.WithStack(NewIllegalOperationError("Only owner/root can change permissions."));
    }

    if (perms == direntInfo.Permissions) {
        return nil;
    }

    direntInfo.Permissions = perms;
    this.cache.CacheDirentPut(direntInfo);

    return nil;
}

func (this *Driver) FetchChildByName(userId identity.UserId, parentId dirent.Id, name string) (*dirent.Dirent, error) {
    _, user, err := this.getUserAndDirent(userId, parentId, true, false, true, false, true);
    if (err != nil) {
        return nil, errors.WithStack(err);
    }

    for _, child := range(this.dirs[parentId]) {
        if (child.Name == name) {
            childGroup, ok := this.groups[child.Group];
            if (!ok) {
                return nil, errors.Errorf("Unable to find the group (%d) for dirent (%s).", int(child.Group), string(child.Id));
            }

            if (!child.CanRead(user, childGroup)) {
                return nil, nil;
            }

            return child, nil;
        }
    }

    return nil, nil;
}
