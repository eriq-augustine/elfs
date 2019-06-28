package identity;

/**
 * Bellow are the semantics of the group permission system.
 *  - All user's have a "usergroup" that share a name with the user.
 *      - This is a group is tied to the user.
 *      - This group cannot be deleted unless the user al being dropped (even by root).
 *      - A user cannot change their usergroup.
 *      - All dirent's created by a user will takeon the usergroup initially (but the group can be later changed).
 *      - Althrough others can be added to a usergroup, the owner should be aware of the potential security implications.
 *  - Only the group owner (or root) can drop a group.
 *      - A dirent whose group gets dropped will fallback to the dirent's owner's usergroup.
 *  - All driver-level operations will leave the filesystem in a consistent state.
 *      - So operations that may seem quick from UNIX (like dropping a group) may take a while since the fs/cache may need to be searched.
 *  - Only the owner/root can add to a group.
 *  - Only the owner/root can promote another member to be owner.
 *      - There can only be one owner at a time.
 *  - The owner, root, or context member may remove the context member from a group.
 *      - A user cannot leave their own usergroup.
 *      - If the owner leaves a group, root assumes ownership.
 */

const (
    ROOT_GROUP_ID = GroupId(0)
    EMPTY_GROUP_ID = GroupId(-1)
)

type GroupId int;

type Group struct {
    Id GroupId
    Name string
    IsUsergroup bool
    Owner UserId
    Members map[UserId]bool
}

func NewGroup(id GroupId, name string, owner UserId, isUsergroup bool) *Group {
    var group Group = Group{
        Id: id,
        Name: name,
        IsUsergroup: isUsergroup,
        Owner: owner,
        Members: map[UserId]bool{owner: true},
    };

    return &group;
}

func (this *Group) HasMember(targetUser UserId) bool {
    return this.Members[targetUser];
}
