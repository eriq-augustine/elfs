package user;

import (
   "math/rand"

   "github.com/eriq-augustine/golog"
   "golang.org/x/crypto/bcrypt"
)

const (
   ROOT_ID = Id(0)
   ROOT_NAME = "root"
)

type Id int;

type User struct {
   Id Id
   Passhash string
   Name string
   Email string
}

func New(id Id, weakhash string, name string, email string) (*User, error) {
   bcryptHash, err := bcrypt.GenerateFromPassword([]byte(weakhash), bcrypt.DefaultCost);
   if (err != nil) {
      golog.ErrorE("Could not generate bcrypt hash", err);
      return nil, err;
   }

   var user User = User{
      Id: id,
      Passhash: string(bcryptHash),
      Name: name,
      Email: email,
   };

   return &user, nil;
}

func NewUserId(otherUsers map[Id]*User) Id {
   var id Id = Id(rand.Int());

   if (otherUsers == nil) {
      return id;
   }

   for {
      _, ok := otherUsers[id];
      if (!ok) {
         break;
      }

      id = Id(rand.Int());
   }

   return id;
}
