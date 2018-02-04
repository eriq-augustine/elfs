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
   "github.com/eriq-augustine/elfs/group"
   "github.com/eriq-augustine/elfs/user"
   "github.com/eriq-augustine/elfs/util"
)

const (
   COMMAND_LOGIN = "login"
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
      RequireLogin: true,
      Variatic: true,
   };

   commands["create"] = commandInfo{
      Name: "create",
      Function: create,
      Args: []commandArg{
         commandArg{"root password", false},
      },
      RequireLogin: false,
      Variatic: false,
   };

   commands["demote"] = commandInfo{
      Name: "demote",
      Function: demote,
      Args: []commandArg{
         commandArg{"group id", false},
         commandArg{"user id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["export"] = commandInfo{
      Name: "export",
      Function: export,
      Args: []commandArg{
         commandArg{"file", false},
         commandArg{"external path", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["groupadd"] = commandInfo{
      Name: "groupadd",
      Function: groupadd,
      Args: []commandArg{
         commandArg{"group name", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["groupdel"] = commandInfo{
      Name: "groupdel",
      Function: groupdel,
      Args: []commandArg{
         commandArg{"group id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["groupjoin"] = commandInfo{
      Name: "groupjoin",
      Function: groupjoin,
      Args: []commandArg{
         commandArg{"group id", false},
         commandArg{"user id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["groupkick"] = commandInfo{
      Name: "groupkick",
      Function: groupkick,
      Args: []commandArg{
         commandArg{"group id", false},
         commandArg{"user id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["grouplist"] = commandInfo{
      Name: "grouplist",
      Function: grouplist,
      Args: []commandArg{},
      RequireLogin: true,
      Variatic: false,
   };

   commands["help"] = commandInfo{
      Name: "help",
      Function: help,
      Args: []commandArg{},
      RequireLogin: false,
      Variatic: false,
   };

   commands["import"] = commandInfo{
      Name: "import",
      Function: importFile,
      Args: []commandArg{
         commandArg{"external file", false},
         commandArg{"parent id", true},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands[COMMAND_LOGIN] = commandInfo{
      Name: COMMAND_LOGIN,
      Function: login,
      Args: []commandArg{
         commandArg{"username", false},
         commandArg{"password", false},
      },
      RequireLogin: false,
      Variatic: false,
   };

   commands["ls"] = commandInfo{
      Name: "ls",
      Function: ls,
      Args: []commandArg{
         commandArg{"dir id", true},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["mkdir"] = commandInfo{
      Name: "mkdir",
      Function: mkdir,
      Args: []commandArg{
         commandArg{"dir name", false},
         commandArg{"parent id", true},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["mv"] = commandInfo{
      Name: "mv",
      Function: move,
      Args: []commandArg{
         commandArg{"target id", false},
         commandArg{"new parent id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["promote"] = commandInfo{
      Name: "promote",
      Function: promote,
      Args: []commandArg{
         commandArg{"group id", false},
         commandArg{"user id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["rename"] = commandInfo{
      Name: "rename",
      Function: rename,
      Args: []commandArg{
         commandArg{"target id", false},
         commandArg{"new name", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["rm"] = commandInfo{
      Name: "rm",
      Function: remove,
      Args: []commandArg{
         commandArg{"-r", true},
         commandArg{"dirent id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["useradd"] = commandInfo{
      Name: "useradd",
      Function: useradd,
      Args: []commandArg{
         commandArg{"username", false},
         commandArg{"password", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["userdel"] = commandInfo{
      Name: "userdel",
      Function: userdel,
      Args: []commandArg{
         commandArg{"username", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["userlist"] = commandInfo{
      Name: "userlist",
      Function: userlist,
      Args: []commandArg{},
      RequireLogin: true,
      Variatic: false,
   };

   commands["chown"] = commandInfo{
      Name: "chown",
      Function: chown,
      Args: []commandArg{
         commandArg{"dirent id", false},
         commandArg{"new owner id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["permadd"] = commandInfo{
      Name: "permadd",
      Function: permissionAdd,
      Args: []commandArg{
         commandArg{"dirent id", false},
         commandArg{"group id", false},
         commandArg{"2|4|6", false},
      },
      RequireLogin: true,
      Variatic: false,
   };

   commands["permdel"] = commandInfo{
      Name: "permdel",
      Function: permissionDelete,
      Args: []commandArg{
         commandArg{"dirent id", false},
         commandArg{"group id", false},
      },
      RequireLogin: true,
      Variatic: false,
   };
}

// Commands

func cat(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var buffer []byte = make([]byte, cipherio.IO_BLOCK_SIZE);

   for _, arg := range(args) {
      // Reset the buffer from the last read.
      buffer = buffer[0:cap(buffer)];

      reader, err := fsDriver.Read(activeUser.Id, dirent.Id(arg));
      if (err != nil) {
         return nil, errors.Wrap(err, "Failed to open fs file for reading: " + arg);
      }

      var done bool = false;
      for (!done) {
         readSize, err := reader.Read(buffer);
         if (err != nil) {
            if (err != io.EOF) {
               return nil, errors.Wrap(err, "Failed to read fs file: " + arg);
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

   return nil, nil;
}

func export(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var source dirent.Id = dirent.Id(args[0]);
   var dest string = args[1];

   fileInfo, err := fsDriver.GetDirent(activeUser.Id, source);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to get dirent for export");
   }

   if (!fileInfo.IsFile) {
      return nil, errors.New("Recursive export is currently not supported.");
   }

   // Check if the external path is a directory.
   // If so, make the target path that directory with the file's current name.
   stat, err := os.Stat(dest);
   if (err == nil && stat.IsDir()) {
      dest = filepath.Join(dest, fileInfo.Name);
   }

   outFile, err := os.Create(dest);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to create outout file for export.");
   }
   defer outFile.Close();

   var buffer []byte = make([]byte, cipherio.IO_BLOCK_SIZE);

   reader, err := fsDriver.Read(activeUser.Id, source);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to open fs file for reading: " + string(source));
   }
   defer reader.Close();

   var done bool = false;
   for (!done) {
      readSize, err := reader.Read(buffer);
      if (err != nil) {
         if (err != io.EOF) {
            return nil, errors.Wrap(err, "Failed to read fs file: " + string(source));
         }

         done = true;
      }

      if (readSize > 0) {
         outFile.Write(buffer[0:readSize]);
      }
   }

   return nil, nil;
}

func create(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   return nil, fsDriver.CreateFilesystem(util.Weakhash(user.ROOT_NAME, args[0]));
}

func help(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var keys []string = make([]string, 0, len(commands));
   for key, _ := range(commands) {
      keys = append(keys, key);
   }

   sort.Strings(keys);

   fmt.Println("Commands:");
   for _, key := range(keys) {
      fmt.Printf("   %s\n", commands[key].Usage());
   }

   return nil, nil;
}

func importFile(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var localPath string = args[0];

   var parent dirent.Id = dirent.ROOT_ID;
   if (len(args) == 2) {
      parent = dirent.Id(args[1]);
   }

   _, err := recursiveImport(fsDriver, activeUser, localPath, parent);
   return nil, errors.WithStack(err);
}

func login(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   authUser, err := fsDriver.UserAuth(args[0], util.Weakhash(args[0], args[1]));
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to authenticate user.");
   }

   return authUser, nil;
}

func ls(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var id dirent.Id = dirent.ROOT_ID;
   if (len(args) == 1) {
      id = dirent.Id(args[0]);
   }

   entries, err := fsDriver.List(activeUser.Id, id);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to list directory: " + string(id));
   }

   var parts []string = make([]string, 0);
   var groups []string = make([]string, 0);

   for _, entry := range(entries) {
      parts = parts[:0];
      groups = parts[:0];

      var direntType string = "D";
      if (entry.IsFile) {
         direntType = "F";
      }

      parts = append(parts, entry.Name, direntType,
            string(entry.Id), fmt.Sprintf("%d", entry.Size), fmt.Sprintf("%d", entry.ModTimestamp), entry.Md5);

      // Get the group permissions.
      for groupId, permission := range(entry.GroupPermissions) {
         var access string = "";

         if (permission.Read) {
            access += "R";
         } else {
            access += "-";
         }

         if (permission.Write) {
            access += "W";
         } else {
            access += "-";
         }

         groups = append(groups, fmt.Sprintf("%s: %s", groupId, access));
      }
      parts = append(parts, fmt.Sprintf("[%s]", strings.Join(groups, ", ")));

      fmt.Println(strings.Join(parts, "\t"));
   }

   return nil, nil;
}

func mkdir(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var name string = args[0];

   var parent dirent.Id = dirent.ROOT_ID;
   if (len(args) == 2) {
      parent = dirent.Id(args[1]);
   }

   id, err := fsDriver.MakeDir(activeUser.Id, name, parent, map[group.Id]group.Permission{});
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to make dir: " + name);
   }

   fmt.Println(id);

   return nil, nil;
}

func move(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var targetId dirent.Id = dirent.Id(args[0]);
   var newParentId dirent.Id = dirent.Id(args[1]);

   return nil, errors.WithStack(fsDriver.Move(activeUser.Id, targetId, newParentId));
}

func rename(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var targetId dirent.Id = dirent.Id(args[0]);

   return nil, errors.WithStack(fsDriver.Rename(activeUser.Id, targetId, args[1]));
}

func remove(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   if (len(args) == 2 && args[0] != "-r") {
      return nil, errors.New(fmt.Sprintf("Unexpected arg (%s), expecting -r", args[0]));
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

   return nil, errors.WithStack(err);
}

func useradd(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   _, err := fsDriver.AddUser(activeUser.Id, args[0], util.Weakhash(args[0], args[1]));
   return nil, errors.Wrap(err, "Failed to add user");
}

func userdel(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   userId, err := strconv.Atoi(args[0]);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to parse user id");
   }

   err = fsDriver.RemoveUser(activeUser.Id, user.Id(userId));
   return nil, errors.Wrap(err, "Failed to remove user");
}

func userlist(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   users := fsDriver.GetUsers();

   for _, user := range(users) {
      fmt.Printf("%s\t%d\n", user.Name, int(user.Id));
   }

   return nil, nil;
}

func demote(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   groupId, err := strconv.Atoi(args[0]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[0]);
   }

   userId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   return nil, errors.WithStack(fsDriver.DemoteUser(activeUser.Id, user.Id(userId), group.Id(groupId)));
}

func groupadd(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   newId, err := fsDriver.AddGroup(activeUser.Id, args[0]);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   fmt.Println(newId);
   return nil, nil;
}

func groupdel(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   groupId, err := strconv.Atoi(args[0]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[0]);
   }

   return nil, errors.WithStack(fsDriver.DeleteGroup(activeUser.Id, group.Id(groupId)));
}

func groupjoin(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   groupId, err := strconv.Atoi(args[0]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[0]);
   }

   userId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   return nil, errors.WithStack(fsDriver.JoinGroup(activeUser.Id, user.Id(userId), group.Id(groupId)));
}

func groupkick(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   groupId, err := strconv.Atoi(args[0]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[0]);
   }

   userId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   return nil, errors.WithStack(fsDriver.KickUser(activeUser.Id, user.Id(userId), group.Id(groupId)));
}

func grouplist(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   groups := fsDriver.GetGroups();

   var parts []string = make([]string, 0);
   for _, group := range(groups) {
      parts = parts[:0];

      parts = append(parts, group.Name);
      parts = append(parts, fmt.Sprintf("%d", int(group.Id)));

      for userId, _ := range(group.Users) {
         var name string;
         if (group.Admins[userId]) {
            name = fmt.Sprintf("%d*", int(userId));
         } else {
            name = fmt.Sprintf("%d", int(userId));
         }

         parts = append(parts, name);
      }

      fmt.Println(strings.Join(parts, "\t"));
   }

   return nil, nil;
}

func promote(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   groupId, err := strconv.Atoi(args[0]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[0]);
   }

   userId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   return nil, errors.WithStack(fsDriver.PromoteUser(activeUser.Id, user.Id(userId), group.Id(groupId)));
}

func chown(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var direntId dirent.Id = dirent.Id(args[0]);

   userId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   return nil, errors.WithStack(fsDriver.ChangeOwner(activeUser.Id, direntId, user.Id(userId)));
}

func permissionAdd(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var direntId dirent.Id = dirent.Id(args[0]);

   groupId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   permission, err := strconv.Atoi(args[2]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[2]);
   }

   if (permission != 2 && permission != 4 && permission != 6) {
      return nil, errors.Errorf("Bad permission number: %d. Use UNIX-style for read and write", permission);
   }

   var read bool = (permission % 4 == 0);
   var write bool = (permission % 2 == 0);

   return nil, errors.WithStack(fsDriver.PutGroupAccess(activeUser.Id, direntId, group.Id(groupId), group.NewPermission(read, write)));
}

func permissionDelete(fsDriver *driver.Driver, activeUser *user.User, args []string) (interface{}, error) {
   var direntId dirent.Id = dirent.Id(args[0]);

   groupId, err := strconv.Atoi(args[1]);
   if (err != nil) {
      return nil, errors.Wrap(err, args[1]);
   }

   return nil, errors.WithStack(fsDriver.RemoveGroupAccess(activeUser.Id, direntId, group.Id(groupId)));
}

// Helpers

func importFileInternal(fsDriver *driver.Driver, activeUser *user.User, path string, parent dirent.Id) (interface{}, error) {
   fileReader, err := os.Open(path);
   if (err != nil) {
      return nil, errors.Wrap(err, path);
   }
   defer fileReader.Close();

   err = fsDriver.Put(activeUser.Id, filepath.Base(path), fileReader, map[group.Id]group.Permission{}, parent);
   if (err != nil) {
      return nil, errors.Wrap(err, path);
   }

   return nil, nil;
}

func recursiveImport(fsDriver *driver.Driver, activeUser *user.User, path string, parent dirent.Id) (interface{}, error) {
   fileInfo, err := os.Stat(path);
   if (err != nil) {
      return nil, errors.Wrap(err, path);
   }

   if (!fileInfo.IsDir()) {
      _, err = importFileInternal(fsDriver, activeUser, path, parent);
      return nil, errors.WithStack(err);
   }

   // First make the actual dir and then import the children.
   newId, err := fsDriver.MakeDir(activeUser.Id, fileInfo.Name(), parent, map[group.Id]group.Permission{});
   if (err != nil) {
      return nil, errors.Wrap(err, path);
   }

   children, err := ioutil.ReadDir(path);
   if (err != nil) {
      return nil, errors.Wrap(err, path);
   }

   for _, child := range(children) {
      _, err = recursiveImport(fsDriver, activeUser, filepath.Join(path, child.Name()), newId);
      if (err != nil) {
         return nil, errors.Wrap(err, path);
      }
   }

   return nil, nil;
}
