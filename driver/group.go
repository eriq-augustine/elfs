package driver;

// Operations dealing with groups in the filesystem.

import (
   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/group"
   "github.com/eriq-augustine/elfs/user"
)

func (this *Driver) AddGroup(contextUser user.Id, name string) (group.Id, error) {
   if (name == "") {
      return group.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot create group with no name."));
   }

   for _, groupInfo := range(this.groups) {
      if (groupInfo.Name == name) {
         return group.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot create group with existing name: " + name));
      }
   }

   newGroup := group.New(this.getNewGroupId(), name, contextUser);

   this.groups[newGroup.Id] = newGroup;
   this.cache.CacheGroupPut(newGroup);

   return newGroup.Id, nil;
}

func (this *Driver) DeleteGroup(contextUser user.Id, groupId group.Id) error {
   groupInfo, ok := this.groups[groupId];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot remove unknown group."));
   }

   if (groupInfo.Id == group.EVERYBODY_ID) {
      return errors.WithStack(NewIllegalOperationError("Cannot remove everybody group"));
   }

   err := this.checkGroupAdminPermissions(contextUser, groupInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Remove this group from the fat.
   this.purgeGroup(groupId);

   delete(this.groups, groupId);
   this.cache.CacheGroupDelete(groupInfo);

   return nil;
}

func (this *Driver) DemoteUser(contextUser user.Id, targetUser user.Id, groupId group.Id) error {
   if (contextUser != user.ROOT_ID) {
      return errors.WithStack(NewIllegalOperationError("Only root can demote users."));
   }

   groupInfo, ok := this.groups[groupId];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot demote in unknown group."));
   }

   _, ok = this.users[targetUser];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Demotion candidate does not exist."));
   }

   if (!groupInfo.Admins[targetUser]) {
      return nil;
   }

   delete(groupInfo.Admins, targetUser);
   this.cache.CacheGroupPut(groupInfo);

   return nil;
}

func (this *Driver) GetGroups() map[group.Id]*group.Group {
   return this.groups;
}

func (this *Driver) JoinGroup(contextUser user.Id, targetUser user.Id, groupId group.Id) error {
   groupInfo, ok := this.groups[groupId];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot join an unknown group."));
   }

   err := this.checkGroupAdminPermissions(contextUser, groupInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   _, ok = this.users[targetUser];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Group join candidate does not exist."));
   }

   if (groupInfo.Users[targetUser]) {
      return nil;
   }

   groupInfo.Users[targetUser] = true;
   this.cache.CacheGroupPut(groupInfo);

   return nil;
}

func (this *Driver) KickUser(contextUser user.Id, targetUser user.Id, groupId group.Id) error {
   groupInfo, ok := this.groups[groupId];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot kick from an unknown group."));
   }

   err := this.checkGroupAdminPermissions(contextUser, groupInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   _, ok = this.users[targetUser];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Kick candidate does not exist."));
   }

   if (!groupInfo.Users[targetUser]) {
      return nil;
   }

   // Only root can kick an admin.
   if (contextUser != user.ROOT_ID && groupInfo.Admins[targetUser]) {
      return errors.WithStack(NewIllegalOperationError("Only root can kick an admin."));
   }

   delete(groupInfo.Users, targetUser);
   this.cache.CacheGroupPut(groupInfo);

   return nil;
}

func (this *Driver) PromoteUser(contextUser user.Id, targetUser user.Id, groupId group.Id) error {
   groupInfo, ok := this.groups[groupId];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot promote in unknown group."));
   }

   err := this.checkGroupAdminPermissions(contextUser, groupInfo);
   if (err != nil) {
      return errors.WithStack(err);
   }

   _, ok = this.users[targetUser];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Promotion candidate does not exist."));
   }

   if (!groupInfo.Users[targetUser]) {
      return errors.WithStack(NewIllegalOperationError("Promotion candidate is not a member of the group."));
   }

   if (groupInfo.Admins[targetUser]) {
      return nil;
   }

   groupInfo.Admins[targetUser] = true;
   this.cache.CacheGroupPut(groupInfo);

   return nil;
}

// Go through the entire FAT and ensure that there are no traces of this group.
func (this *Driver) purgeGroup(groupId group.Id) {
   for _, direntInfo := range(this.fat) {
      _, ok := direntInfo.GroupPermissions[groupId];
      if (ok) {
         delete(direntInfo.GroupPermissions, groupId);
         this.cache.CacheDirentPut(direntInfo);
      }
   }
}
