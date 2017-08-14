package driver;

// Operations dealing with users in the filesystem.

import (
   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) AddUser(contextUser user.Id, name string, weakhash string) (user.Id, error) {
   if (contextUser != user.ROOT_ID) {
      return user.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Only root can add users."));
   }

   if (name == "") {
      return user.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with no name."));
   }

   if (weakhash == "") {
      return user.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with empty password."));
   }

   for _, userInfo := range(this.users) {
      if (userInfo.Name == name) {
         return user.EMPTY_ID, errors.WithStack(NewIllegalOperationError("Cannot create user with existing name."));
      }
   }

   newUser, err := user.New(this.getNewUserId(), weakhash, name);
   if (err != nil) {
      return user.EMPTY_ID, errors.Wrap(err, "Failed to create new user.");
   }

   this.users[newUser.Id] = newUser;
   this.cache.CacheUserPut(newUser);

   // Add the user to the everybody group.
   this.groups[group.EVERYBODY_ID].Users[newUser.Id] = true;
   this.cache.CacheGroupPut(this.groups[group.EVERYBODY_ID]);

   return newUser.Id, nil;
}

func (this *Driver) GetUsers(contextUser user.Id) (map[user.Id]*user.User, error) {
   if (contextUser != user.ROOT_ID) {
      return nil, errors.WithStack(NewIllegalOperationError("Only root can list users."));
   }

   return this.users, nil;
}

func (this *Driver) RemoveUser(contextUser user.Id, targetId user.Id) error {
   if (contextUser != user.ROOT_ID) {
      return errors.WithStack(NewIllegalOperationError("Only root can delete users."));
   }

   if (targetId == user.ROOT_ID) {
      return errors.WithStack(NewIllegalOperationError("Cannot remove root user."));
   }

   targetUser, ok := this.users[targetId];
   if (!ok) {
      return errors.WithStack(NewIllegalOperationError("Cannot delete unknown user."));
   }

   // Transfer ownership of all resources to root.
   this.transferOwnership(targetUser.Id, user.ROOT_ID);
   this.purgeFromGroups(targetUser.Id);

   delete(this.users, targetUser.Id);
   this.cache.CacheUserDelete(targetUser);

   return nil;
}

func (this *Driver) UserAuth(name string, weakhash string) (*user.User, error) {
   var targetUser *user.User = nil;
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

// Transfer owneership of all files from one user to another.
func (this *Driver) transferOwnership(oldUser user.Id, newUser user.Id) {
   // TODO(eriq)
}

// Remove all traces of a user from all groups.
func (this *Driver) purgeFromGroups(targetUser user.Id) {
   // TODO(eriq)
}
