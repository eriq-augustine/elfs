package main;

import (
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/cipherio"
    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/driver"
    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/util"
)

const (
    COMMAND_QUIT = "quit"
)

var commands map[string]commandInfo;

func init() {
    commands = make(map[string]commandInfo);

    commands["cat"] = commandInfo{
        Name: "cat",
        Function: cat,
        Args: []commandArg{
            commandArg{"file", false},
        },
        Variatic: true,
    };

    commands["export"] = commandInfo{
        Name: "export",
        Function: export,
        Args: []commandArg{
            commandArg{"file", false},
            commandArg{"external path", false},
        },
        Variatic: false,
    };

    commands["groupadd"] = commandInfo{
        Name: "groupadd",
        Function: groupadd,
        Args: []commandArg{
            commandArg{"group name", false},
        },
        Variatic: false,
    };

    commands["groupdel"] = commandInfo{
        Name: "groupdel",
        Function: groupdel,
        Args: []commandArg{
            commandArg{"group id", false},
        },
        Variatic: false,
    };

    commands["groupjoin"] = commandInfo{
        Name: "groupjoin",
        Function: groupjoin,
        Args: []commandArg{
            commandArg{"group id", false},
            commandArg{"user id", false},
        },
        Variatic: false,
    };

    commands["groupkick"] = commandInfo{
        Name: "groupkick",
        Function: groupkick,
        Args: []commandArg{
            commandArg{"group id", false},
            commandArg{"user id", false},
        },
        Variatic: false,
    };

    commands["grouplist"] = commandInfo{
        Name: "grouplist",
        Function: grouplist,
        Args: []commandArg{},
        Variatic: false,
    };

    commands["help"] = commandInfo{
        Name: "help",
        Function: help,
        Args: []commandArg{},
        Variatic: false,
    };

    commands["import"] = commandInfo{
        Name: "import",
        Function: importFile,
        Args: []commandArg{
            commandArg{"external file", false},
            commandArg{"parent id", true},
        },
        Variatic: false,
    };

    commands["ls"] = commandInfo{
        Name: "ls",
        Function: ls,
        Args: []commandArg{
            commandArg{"dir id", true},
        },
        Variatic: false,
    };

    commands["mkdir"] = commandInfo{
        Name: "mkdir",
        Function: mkdir,
        Args: []commandArg{
            commandArg{"dir name", false},
            commandArg{"parent id", true},
        },
        Variatic: false,
    };

    commands["mv"] = commandInfo{
        Name: "mv",
        Function: move,
        Args: []commandArg{
            commandArg{"target id", false},
            commandArg{"new parent id", false},
        },
        Variatic: false,
    };

    commands["promote"] = commandInfo{
        Name: "promote",
        Function: promote,
        Args: []commandArg{
            commandArg{"group id", false},
            commandArg{"user id", false},
        },
        Variatic: false,
    };

    commands["rename"] = commandInfo{
        Name: "rename",
        Function: rename,
        Args: []commandArg{
            commandArg{"target id", false},
            commandArg{"new name", false},
        },
        Variatic: false,
    };

    commands["rm"] = commandInfo{
        Name: "rm",
        Function: remove,
        Args: []commandArg{
            commandArg{"-r", true},
            commandArg{"dirent id", false},
        },
        Variatic: false,
    };

    commands["useradd"] = commandInfo{
        Name: "useradd",
        Function: useradd,
        Args: []commandArg{
            commandArg{"username", false},
            commandArg{"password", false},
        },
        Variatic: false,
    };

    commands["userdel"] = commandInfo{
        Name: "userdel",
        Function: userdel,
        Args: []commandArg{
            commandArg{"username", false},
        },
        Variatic: false,
    };

    commands["userlist"] = commandInfo{
        Name: "userlist",
        Function: userlist,
        Args: []commandArg{},
        Variatic: false,
    };

    commands["chown"] = commandInfo{
        Name: "chown",
        Function: chown,
        Args: []commandArg{
            commandArg{"-r", true},
            commandArg{"owner id", false},
            commandArg{"group id", false},
            commandArg{"dirent id", false},
        },
        Variatic: false,
    };

    commands["chmod"] = commandInfo{
        Name: "chmod",
        Function: chmod,
        Args: []commandArg{
            commandArg{"-r", true},
            commandArg{"UNIX permissions", false},
            commandArg{"dirent id", false},
        },
        Variatic: false,
    };
}

// Commands

func cat(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var buffer []byte = make([]byte, cipherio.IO_BLOCK_SIZE);

    for _, arg := range(args) {
        // Reset the buffer from the last read.
        buffer = buffer[0:cap(buffer)];

        reader, err := fsDriver.Read(activeUser.Id, dirent.Id(arg));
        if (err != nil) {
            return errors.Wrap(err, "Failed to open fs file for reading: " + arg);
        }

        var done bool = false;
        for (!done) {
            readSize, err := reader.Read(buffer);
            if (err != nil) {
                if (err != io.EOF) {
                    return errors.Wrap(err, "Failed to read fs file: " + arg);
                }

                done = true;
            }

            if (readSize > 0) {
                fmt.Print(string(buffer[0:readSize]));
            }
        }

        fmt.Println("");
        reader.Close();
    }

    return nil;
}

func export(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var source dirent.Id = dirent.Id(args[0]);
    var dest string = args[1];

    fileInfo, err := fsDriver.GetDirent(activeUser.Id, source);
    if (err != nil) {
        return errors.Wrap(err, "Failed to get dirent for export");
    }

    if (!fileInfo.IsFile) {
        return errors.New("Recursive export is currently not supported.");
    }

    // Check if the external path is a directory.
    // If so, make the target path that directory with the file's current name.
    stat, err := os.Stat(dest);
    if (err == nil && stat.IsDir()) {
        dest = filepath.Join(dest, fileInfo.Name);
    }

    outFile, err := os.Create(dest);
    if (err != nil) {
        return errors.Wrap(err, "Failed to create outout file for export.");
    }
    defer outFile.Close();

    var buffer []byte = make([]byte, cipherio.IO_BLOCK_SIZE);

    reader, err := fsDriver.Read(activeUser.Id, source);
    if (err != nil) {
        return errors.Wrap(err, "Failed to open fs file for reading: " + string(source));
    }
    defer reader.Close();

    var done bool = false;
    for (!done) {
        readSize, err := reader.Read(buffer);
        if (err != nil) {
            if (err != io.EOF) {
                return errors.Wrap(err, "Failed to read fs file: " + string(source));
            }

            done = true;
        }

        if (readSize > 0) {
            outFile.Write(buffer[0:readSize]);
        }
    }

    return nil;
}

func help(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var keys []string = make([]string, 0, len(commands));
    for key, _ := range(commands) {
        keys = append(keys, key);
    }

    sort.Strings(keys);

    fmt.Println("Commands:");
    for _, key := range(keys) {
        fmt.Printf("    %s\n", commands[key].Usage());
    }

    // Print quit specially, since it caught higher up.
    fmt.Printf("        %s\n", COMMAND_QUIT);

    return nil;
}

func importFile(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var localPath string = args[0];

    var parent dirent.Id = dirent.ROOT_ID;
    if (len(args) == 2) {
        parent = dirent.Id(args[1]);
    }

    return errors.WithStack(recursiveImport(fsDriver, activeUser, localPath, parent));
}

func ls(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var id dirent.Id = dirent.ROOT_ID;
    if (len(args) == 1) {
        id = dirent.Id(args[0]);
    }

    entries, err := fsDriver.List(activeUser.Id, id);
    if (err != nil) {
        return errors.Wrap(err, "Failed to list directory: " + string(id));
    }

    for _, entry := range(entries) {
        var parts []string = make([]string, 0, 8);

        var direntType string = "d";
        if (entry.IsFile) {
            direntType = "-";
        }

        parts = append(parts, (direntType + entry.Permissions.String()), string(entry.Owner), string(entry.Group),
                fmt.Sprintf("%d", entry.Size), fmt.Sprintf("%d", entry.ModTimestamp), entry.Md5,
                string(entry.Id), entry.Name);

        fmt.Println(strings.Join(parts, "\t"));
    }

    return nil;
}

func mkdir(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var name string = args[0];

    var parent dirent.Id = dirent.ROOT_ID;
    if (len(args) == 2) {
        parent = dirent.Id(args[1]);
    }

    id, err := fsDriver.MakeDir(activeUser.Id, name, parent);
    if (err != nil) {
        return errors.Wrap(err, "Failed to make dir: " + name);
    }

    fmt.Println(id);

    return nil;
}

func move(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var targetId dirent.Id = dirent.Id(args[0]);
    var newParentId dirent.Id = dirent.Id(args[1]);

    return errors.WithStack(fsDriver.Move(activeUser.Id, targetId, newParentId));
}

func rename(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    var targetId dirent.Id = dirent.Id(args[0]);

    return errors.WithStack(fsDriver.Rename(activeUser.Id, targetId, args[1]));
}

func remove(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    if (len(args) == 2 && args[0] != "-r") {
        return errors.New(fmt.Sprintf("Unexpected arg (%s), expecting -r", args[0]));
    }

    var isFile = true;
    if (len(args) == 2) {
        isFile = false;
        args = args[1:];
    }

    var direntId dirent.Id = dirent.Id(args[0]);

    var err error = nil;
    if (isFile) {
        err = fsDriver.RemoveFile(activeUser.Id, direntId);
    } else {
        err = fsDriver.RemoveDir(activeUser.Id, direntId);
    }

    return errors.WithStack(err);
}

func useradd(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    _, err := fsDriver.AddUser(activeUser.Id, args[0], util.Weakhash(args[0], args[1]));
    return errors.Wrap(err, "Failed to add user");
}

func userdel(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    userId, err := strconv.Atoi(args[0]);
    if (err != nil) {
        return errors.Wrap(err, "Failed to parse user id");
    }

    err = fsDriver.RemoveUser(activeUser.Id, identity.UserId(userId));
    return errors.Wrap(err, "Failed to remove user");
}

func userlist(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    users := fsDriver.GetUsers();

    for _, user := range(users) {
        fmt.Printf("%s\t%d\n", user.Name, int(user.Id));
    }

    return nil;
}

func groupadd(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    newId, err := fsDriver.AddGroup(activeUser.Id, args[0]);
    if (err != nil) {
        return errors.WithStack(err);
    }

    fmt.Println(newId);
    return nil;
}

func groupdel(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    groupId, err := strconv.Atoi(args[0]);
    if (err != nil) {
        return errors.Wrap(err, args[0]);
    }

    return errors.WithStack(fsDriver.DeleteGroup(activeUser.Id, identity.GroupId(groupId)));
}

func groupjoin(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    groupId, err := strconv.Atoi(args[0]);
    if (err != nil) {
        return errors.Wrap(err, args[0]);
    }

    userId, err := strconv.Atoi(args[1]);
    if (err != nil) {
        return errors.Wrap(err, args[1]);
    }

    return errors.WithStack(fsDriver.JoinGroup(activeUser.Id, identity.UserId(userId), identity.GroupId(groupId)));
}

func groupkick(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    groupId, err := strconv.Atoi(args[0]);
    if (err != nil) {
        return errors.Wrap(err, args[0]);
    }

    userId, err := strconv.Atoi(args[1]);
    if (err != nil) {
        return errors.Wrap(err, args[1]);
    }

    return errors.WithStack(fsDriver.KickUser(activeUser.Id, identity.UserId(userId), identity.GroupId(groupId)));
}

func grouplist(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    groups := fsDriver.GetGroups();

    var parts []string = make([]string, 0);
    for _, group := range(groups) {
        parts = parts[:0];

        parts = append(parts, group.Name);
        parts = append(parts, fmt.Sprintf("%d", int(group.Id)));
        parts = append(parts, fmt.Sprintf("%d*", int(group.Owner)));

        for userId, _ := range(group.Members) {
            parts = append(parts, fmt.Sprintf("%d", int(userId)));
        }

        fmt.Println(strings.Join(parts, "\t"));
    }

    return nil;
}

func promote(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    groupId, err := strconv.Atoi(args[0]);
    if (err != nil) {
        return errors.Wrap(err, args[0]);
    }

    userId, err := strconv.Atoi(args[1]);
    if (err != nil) {
        return errors.Wrap(err, args[1]);
    }

    return errors.WithStack(fsDriver.PromoteUser(activeUser.Id, identity.UserId(userId), identity.GroupId(groupId)));
}

func chown(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    if (len(args) == 4 && args[0] != "-r") {
        return errors.New(fmt.Sprintf("Unexpected arg (%s), expecting -r", args[0]));
    }

    var recurse = false;
    if (len(args) == 4) {
        recurse = true;
        args = args[1:];
    }

    userId, err := strconv.Atoi(args[0]);
    if (err != nil) {
        return errors.Wrap(err, args[0]);
    }

    groupId, err := strconv.Atoi(args[1]);
    if (err != nil) {
        return errors.Wrap(err, args[1]);
    }

    direntInfo, err := fsDriver.GetDirent(activeUser.Id, dirent.Id(args[2]));
    if (err != nil) {
        return errors.Wrap(err, args[2]);
    }

    return errors.WithStack(chownHelper(fsDriver, activeUser, direntInfo, identity.UserId(userId), identity.GroupId(groupId), recurse));
}

func chmod(fsDriver *driver.Driver, activeUser *identity.User, args []string) (error) {
    if (len(args) == 3 && args[0] != "-r") {
        return errors.New(fmt.Sprintf("Unexpected arg (%s), expecting -r", args[0]));
    }

    var recurse = false;
    if (len(args) == 3) {
        recurse = true;
        args = args[1:];
    }

    perms, err := dirent.PermissionsFromString(args[0]);
    if (err != nil) {
        return errors.Wrap(err, args[0]);
    }

    direntInfo, err := fsDriver.GetDirent(activeUser.Id, dirent.Id(args[1]));
    if (err != nil) {
        return errors.Wrap(err, args[1]);
    }

    return errors.WithStack(chmodHelper(fsDriver, activeUser, direntInfo, perms, recurse));
}

// Helpers

func chownHelper(fsDriver *driver.Driver, activeUser *identity.User, direntInfo *dirent.Dirent, userId identity.UserId, groupId identity.GroupId, recurse bool) error {
    err := fsDriver.ChangeOwner(activeUser.Id, direntInfo.Id, userId);
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = fsDriver.ChangeGroup(activeUser.Id, direntInfo.Id, groupId);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (recurse && !direntInfo.IsFile) {
        children, err := fsDriver.List(activeUser.Id, direntInfo.Id);
        if (err != nil) {
            return errors.WithStack(err);
        }

        for _, child := range(children) {
            err = chownHelper(fsDriver, activeUser, child, userId, groupId, true);
            if (err != nil) {
                return errors.WithStack(err);
            }
        }
    }

    return nil;
}

func chmodHelper(fsDriver *driver.Driver, activeUser *identity.User, direntInfo *dirent.Dirent, perms dirent.Permissions, recurse bool) error {
    err := fsDriver.ChangePermissions(activeUser.Id, direntInfo.Id, perms);
    if (err != nil) {
        return errors.WithStack(err);
    }

    if (recurse && !direntInfo.IsFile) {
        children, err := fsDriver.List(activeUser.Id, direntInfo.Id);
        if (err != nil) {
            return errors.WithStack(err);
        }

        for _, child := range(children) {
            err = chmodHelper(fsDriver, activeUser, child, perms, true);
            if (err != nil) {
                return errors.WithStack(err);
            }
        }
    }

    return nil;
}

func importFileInternal(fsDriver *driver.Driver, activeUser *identity.User, path string, parent dirent.Id) (error) {
    fileReader, err := os.Open(path);
    if (err != nil) {
        return errors.Wrap(err, path);
    }
    defer fileReader.Close();

    _, err = fsDriver.Put(activeUser.Id, filepath.Base(path), fileReader, parent);
    if (err != nil) {
        return errors.Wrap(err, path);
    }

    return nil;
}

func recursiveImport(fsDriver *driver.Driver, activeUser *identity.User, path string, parent dirent.Id) (error) {
    fileInfo, err := os.Stat(path);
    if (err != nil) {
        return errors.Wrap(err, path);
    }

    if (!fileInfo.IsDir()) {
        return errors.WithStack(importFileInternal(fsDriver, activeUser, path, parent))
    }

    // First make the actual dir and then import the children.
    newId, err := fsDriver.MakeDir(activeUser.Id, fileInfo.Name(), parent);
    if (err != nil) {
        return errors.Wrap(err, path);
    }

    children, err := ioutil.ReadDir(path);
    if (err != nil) {
        return errors.Wrap(err, path);
    }

    for _, child := range(children) {
        err = recursiveImport(fsDriver, activeUser, filepath.Join(path, child.Name()), newId);
        if (err != nil) {
            return errors.Wrap(err, path);
        }
    }

    return nil;
}
