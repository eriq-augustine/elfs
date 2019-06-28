package driver;

// Operations dealing with users in the filesystem.

import (
    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/identity"
)

func (this *Driver) AddUser(contextUser identity.UserId, name string, weakhash string) (identity.UserId, error) {
    if (contextUser != identity.ROOT_USER_ID) {
        return identity.EMPTY_USER_ID, errors.WithStack(NewIllegalOperationError("Only root can add users."));
    }

    if (name == "") {
        return identity.EMPTY_USER_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with no name."));
    }

    if (weakhash == "") {
        return identity.EMPTY_USER_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with empty password."));
    }

    for _, userInfo := range(this.users) {
        if (userInfo.Name == name) {
            return identity.EMPTY_USER_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with existing name."));
        }
    }

    for _, groupInfo := range(this.groups) {
        if (groupInfo.Name == name) {
            return identity.EMPTY_USER_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with same name as existing group (conflicts with usergroups)."));
        }
    }

    newUser, newGroup, err := identity.NewUser(this.getNewUserId(), name, weakhash, this.getNewGroupId());
    if (err != nil) {
        return identity.EMPTY_USER_ID, errors.Wrap(err, "Failed to create new user.");
    }

    this.users[newUser.Id] = newUser;
    this.groups[newGroup.Id] = newGroup;

    this.cache.CacheUserPut(newUser);
    this.cache.CacheGroupPut(newGroup);

    return newUser.Id, nil;
}

func (this *Driver) GetUsers() map[identity.UserId]*identity.User {
    return this.users;
}

func (this *Driver) RemoveUser(contextUser identity.UserId, targetId identity.UserId) error {
    if (contextUser != identity.ROOT_USER_ID) {
        return errors.WithStack(NewIllegalOperationError("Only root can delete users."));
    }

    if (targetId == identity.ROOT_USER_ID) {
        return errors.WithStack(NewIllegalOperationError("Cannot remove root user."));
    }

    targetUser, ok := this.users[targetId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot delete unknown user."));
    }

    targetUsergroup, ok := this.groups[targetUser.Usergroup];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Unable to find usergroup."));
    }

    // Transfer ownership of all resources to root.
    this.transferOwnership(targetUser, this.users[identity.ROOT_USER_ID]);
    this.purgeFromGroups(targetUser.Id);

    // Officially delete the usergroup and user.
    delete(this.groups, targetUsergroup.Id);
    delete(this.users, targetUser.Id);

    this.cache.CacheGroupDelete(targetUsergroup);
    this.cache.CacheUserDelete(targetUser);

    // Because this can cause a lot of cache churn (if this user owned a lot),
    // sync the cache.
    this.SyncToDisk(true);

    return nil;
}

func (this *Driver) UserAuth(name string, weakhash string) (*identity.User, error) {
    var targetUser *identity.User = nil;
    for _, userInfo := range(this.users) {
        if (userInfo.Name == name) {
            targetUser = userInfo;
            break;
        }
    }

    if (targetUser == nil) {
        return nil, errors.WithStack(NewAuthError("Cannot find user to auth"));
    }

    if (targetUser.Auth(weakhash)) {
        return targetUser, nil;
    }

    return nil, errors.WithStack(NewAuthError("Failed to auth user."));
}

// Transfer ownership of all dirents and groups from one user to another.
// No verification is performed.
func (this *Driver) transferOwnership(oldUser *identity.User, newUser *identity.User) {
    for _, entry := range(this.fat) {
        if (entry.Owner == oldUser.Id) {
            entry.Owner = newUser.Id;
        }

        if (entry.Group == oldUser.Usergroup) {
            entry.Group = newUser.Usergroup;
        }
    }

    for _, group := range(this.groups) {
        if (group.Owner == oldUser.Id) {
            group.Owner = newUser.Id;
        }
    }
}

// Remove all traces of a user from all groups.
func (this *Driver) purgeFromGroups(userId identity.UserId) {
    for _, group := range(this.groups) {
        // We will skip the usergroup, since it will be deleted soon.
        if (group.Owner == userId && group.IsUsergroup) {
            continue;
        }

        delete(group.Members, userId);
    }
}
