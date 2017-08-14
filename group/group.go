package group;

import (
   "github.com/eriq-augustine/s3efs/user"
)

const (
   EMPTY_ID = Id(-1)
   EVERYBODY_ID = Id(0)
   EVERYBODY_NAME = "everybody"
)

type Id int;

type Permission struct {
   Read bool
   Write bool
}

// User permissions.
// The admins are a subset of users.
type Group struct {
   Id Id
   Name string
   Admins map[user.Id]bool  // [userId] = true
   Users map[user.Id]bool  // [userId] = true
}

func New(id Id, name string, owner user.Id) *Group {
   var group Group = Group{
      Id: id,
      Name: name,
      Admins: map[user.Id]bool{},
      Users: map[user.Id]bool{},
   };

   group.Admins[owner] = true;
   group.Users[owner] = true;

   return &group;
}

func NewPermission(read bool, write bool) Permission {
   return Permission{
      Read: read,
      Write: write,
   };
}
