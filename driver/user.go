package driver;

// Operations dealing with users in the filesystem.

import (
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) Useradd(name string, email string, passhash string) (user.Id, error) {
   return -1, nil;
}

func (this *Driver) Userdel(user user.Id) error {
   return nil;
}
