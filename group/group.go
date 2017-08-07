package group;

import (
   "math/rand"

   "github.com/emirpasic/gods/sets"
   "github.com/emirpasic/gods/sets/hashset"

   "github.com/eriq-augustine/s3efs/user"
)

const (
   EVERYBODY_ID = Id(0)
   EVERYBODY_NAME = "everybody"
)

type Id int;

type Permission struct {
   GroupId Id
   Read bool
   Write bool
}

// User permissions.
// The admins are a subset of users.
type Group struct {
   Id Id
   Name string
   Admins sets.Set  // Set of user.Id
   Users sets.Set  // Set of user.Id
}

func New(id Id, name string, owner user.Id) *Group {
   var group Group = Group{
      Id: id,
      Name: name,
      Admins: hashset.New(),
      Users: hashset.New(),
   };

   group.Admins.Add(owner);
   return &group;
}

func NewGroupId(otherGroups map[Id]*Group) Id {
   var id Id = Id(rand.Int());

   if (otherGroups == nil) {
      return id;
   }

   for {
      _, ok := otherGroups[id];
      if (!ok) {
         break;
      }

      id = Id(rand.Int());
   }

   return id;
}

func NewPermission(id Id, read bool, write bool) Permission {
   return Permission{
      GroupId: id,
      Read: read,
      Write: write,
   };
}
