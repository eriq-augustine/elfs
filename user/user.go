package user;

import (
   "github.com/eriq-augustine/golog"
   "golang.org/x/crypto/bcrypt"
)

const (
   EMPTY_ID = Id(-1)
   ROOT_ID = Id(0)
   ROOT_NAME = "root"
)

type Id int;

type User struct {
   Id Id
   Passhash string
   Name string
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
   };

   return &user, nil;
}

func (this *User) Auth(weakhash string) bool {
   err := bcrypt.CompareHashAndPassword([]byte(this.Passhash), []byte(weakhash));
   return err == nil;
}
