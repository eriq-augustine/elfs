package main

// fuseDirent will act as both nodes and handles.

import (
    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/driver"
    "github.com/eriq-augustine/elfs/identity"
)

type fuseDirent struct {
    dirent *dirent.Dirent
    driver *driver.Driver
    user *identity.User
}
