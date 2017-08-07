package driver;

// Operations dealing with groups in the filesystem.

import (
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) Groupadd(name string, owner user.Id) (int, error) {
   return -1, nil;
}

func (this *Driver) Groupdel(group group.Id) error {
   return nil;
}

func (this *Driver) JoinGroup(user user.Id, group group.Id) error {
   return nil;
}

func (this *Driver) PromoteUser(user user.Id, group group.Id) error {
   return nil;
}

func (this *Driver) DemoteUser(user user.Id, group group.Id) error {
   return nil;
}
