package driver;

// Operations dealing with groups in the filesystem.

import (
    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/identity"
)

func (this *Driver) GetGroups() map[identity.GroupId]*identity.Group {
    return this.groups;
}

func (this *Driver) AddGroup(contextUser identity.UserId, name string) (identity.GroupId, error) {
    if (name == "") {
        return identity.EMPTY_GROUP_ID, errors.WithStack(NewIllegalOperationError("Cannot create group with no name."));
    }

    for _, groupInfo := range(this.groups) {
        if (groupInfo.Name == name) {
            return identity.EMPTY_GROUP_ID, errors.WithStack(NewIllegalOperationError("Cannot create group with existing name: " + name));
        }
    }

    newGroup := identity.NewGroup(this.getNewGroupId(), name, contextUser, false);

    this.groups[newGroup.Id] = newGroup;
    this.cache.CacheGroupPut(newGroup);

    return newGroup.Id, nil;
}

func (this *Driver) DeleteGroup(contextUser identity.UserId, groupId identity.GroupId) error {
    groupInfo, ok := this.groups[groupId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot remove unknown group."));
    }

    if (groupInfo.IsUsergroup) {
        return errors.WithStack(NewIllegalOperationError("Cannot remove usergroup (must remove user instead)."));
    }

    // Only the group's owner (or root) can remove it.
    if (contextUser != groupInfo.Owner || contextUser != identity.ROOT_USER_ID) {
        return errors.WithStack(NewIllegalOperationError("Only owner or root can remove a group."));
    }

    // Remove this group from the fat.
    this.purgeGroup(groupId);

    delete(this.groups, groupId);
    this.cache.CacheGroupDelete(groupInfo);

    return nil;
}

func (this *Driver) JoinGroup(contextUser identity.UserId, targetUser identity.UserId, groupId identity.GroupId) error {
    groupInfo, ok := this.groups[groupId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot join an unknown group."));
    }

    _, ok = this.users[targetUser];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Group join candidate does not exist."));
    }

    // Only the owner or root can add people to groups.
    if (contextUser != groupInfo.Owner || contextUser != identity.ROOT_USER_ID) {
        return errors.WithStack(NewIllegalOperationError("Only owner or root can add to a group."));
    }

    if (groupInfo.HasMember(targetUser)) {
        return nil;
    }

    groupInfo.Members[targetUser] = true;
    this.cache.CacheGroupPut(groupInfo);

    return nil;
}

func (this *Driver) KickUser(contextUser identity.UserId, targetUser identity.UserId, groupId identity.GroupId) error {
    groupInfo, ok := this.groups[groupId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot kick from an unknown group."));
    }

    _, ok = this.users[targetUser];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Kick candidate does not exist."));
    }

    if (targetUser == groupInfo.Owner && groupInfo.IsUsergroup) {
        return errors.WithStack(NewIllegalOperationError("Cannot kick the owner of a usergroup."));
    }

    // Only the owner or root can add people to groups.
    if (contextUser != groupInfo.Owner || contextUser != identity.ROOT_USER_ID || contextUser == targetUser) {
        return errors.WithStack(NewIllegalOperationError("Only owner, root, or self can kick from a group."));
    }

    if (!groupInfo.Members[targetUser]) {
        return nil;
    }

    // If the owner was kicked, root becomes the new owner.
    // However, usergroups cannot kick/change owners.
    if (targetUser == groupInfo.Owner) {
        groupInfo.Owner = identity.ROOT_USER_ID;
    }

    delete(groupInfo.Members, targetUser);
    this.cache.CacheGroupPut(groupInfo);

    return nil;
}

// Promote a user to be the owner of a group.
func (this *Driver) PromoteUser(contextUser identity.UserId, targetUser identity.UserId, groupId identity.GroupId) error {
    groupInfo, ok := this.groups[groupId];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Cannot promote in unknown group."));
    }

    if (groupInfo.Owner == targetUser) {
        return nil;
    }

    // Usergroups cannot have a different owner.
    if (groupInfo.IsUsergroup) {
        return errors.WithStack(NewIllegalOperationError("Usergroups cannot have a different owner."));
    }

    _, ok = this.users[targetUser];
    if (!ok) {
        return errors.WithStack(NewIllegalOperationError("Promotion candidate does not exist."));
    }

    // Only the owner or root can promote.
    if (contextUser != groupInfo.Owner || contextUser != identity.ROOT_USER_ID) {
        return errors.WithStack(NewIllegalOperationError("Only owner or root can add to a group."));
    }

    // The candidate must already be in the group.
    if (!groupInfo.Members[targetUser]) {
        return errors.WithStack(NewIllegalOperationError("Promotion candidate is not a member of the group."));
    }

    groupInfo.Owner = targetUser;
    this.cache.CacheGroupPut(groupInfo);

    return nil;
}

// Go through the entire FAT and ensure that there are no traces of this group.
// When a group is removed, the dirent get's the owner's usergroup.
func (this *Driver) purgeGroup(groupId identity.GroupId) {
    for _, direntInfo := range(this.fat) {
        if (direntInfo.Group == groupId) {
            direntInfo.Group = this.users[direntInfo.Owner].Usergroup;
        }
    }
}
