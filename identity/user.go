package identity;

import (
    "github.com/eriq-augustine/golog"
    "golang.org/x/crypto/bcrypt"
)

const (
    ROOT_NAME = "root"
    ROOT_USER_ID = UserId(0)
    EMPTY_USER_ID = UserId(-1)
)

type UserId int;

type User struct {
    Id UserId
    Passhash string
    Name string
    Usergroup GroupId
}

func NewUser(
        userId UserId, name string, weakhash string,
        usergroupId GroupId) (*User, *Group, error) {
    // Check that the hash is clean.
    bcryptHash, err := bcrypt.GenerateFromPassword([]byte(weakhash), bcrypt.DefaultCost);
    if (err != nil) {
        golog.ErrorE("Could not generate bcrypt hash", err);
        return nil, nil, err;
    }

    // Make the usergroup.
    var usergroup *Group = NewGroup(usergroupId, name, userId, true);

    var user User = User{
        Id: userId,
        Passhash: string(bcryptHash),
        Name: name,
        Usergroup: usergroup.Id,
    };

    return &user, usergroup, nil;
}

func (this *User) Auth(weakhash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(this.Passhash), []byte(weakhash));
    return err == nil;
}
