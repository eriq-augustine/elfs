package main;

import (
   "bufio"
   "encoding/hex"
   "flag"
   "fmt"
   "io"
   "os"
   "path/filepath"
   "strings"

   "github.com/pkg/errors"
   shellquote "github.com/kballard/go-shellquote"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/driver"
   "github.com/eriq-augustine/s3efs/driver/local"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
   "github.com/eriq-augustine/s3efs/util"
)

// TODO(eriq): Don't allow create as a command.
//  If dir/fat does not initially exist, prompt for root info and create.

// Params: (invocation name, fs driver, args (not including invocation)).
type commandFunction func(string, *driver.Driver, []string) error;

const (
   COMMAND_QUIT = "quit"
   COMMAND_LOGIN = "login"
)

var commands map[string]commandFunction;
var activeUser *user.User;

func init() {
   activeUser = nil;

   commands = make(map[string]commandFunction);

   commands["cat"] = cat;
   commands["create"] = create;
   commands["help"] = help;
   commands["import"] = importFile;
   commands[COMMAND_LOGIN] = login;
   commands["ls"] = ls;
   commands["mkdir"] = mkdir;
}

func main() {
   key, iv, path, err := parseArgs();
   if (err != nil) {
      flag.Usage();
      fmt.Printf("Error parsing args: %+v\n", err);
      return;
   }

   fsDriver, err := local.NewDriver(key, iv, path);
   if (err != nil) {
      panic(fmt.Sprintf("%+v", errors.Wrap(err, "Failed to get local driver")));
   }

   // Try to init the filesystem from any existing metadata.
   err = fsDriver.SyncFromDisk();
   if (err != nil && errors.Cause(err) != nil && !os.IsNotExist(errors.Cause(err))) {
      fmt.Printf("Error parsing existing metadata: %+v\n", err);
      return;
   }

   var scanner *bufio.Scanner = bufio.NewScanner(os.Stdin);
   for {
      if (activeUser == nil) {
         fmt.Printf("> ");
      } else {
         fmt.Printf("%s > ", activeUser.Name);
      }

      if (!scanner.Scan()) {
         break;
      }

      var command string = strings.TrimSpace(scanner.Text());

      if (command == "") {
         continue;
      }

      if (strings.HasPrefix(command, COMMAND_QUIT)) {
         break;
      }

      err = processCommand(fsDriver, command);
      if (err != nil) {
         fmt.Println("Failed to run command:");
         fmt.Printf("%+v\n", err);
      }
   }

   fsDriver.Close();
}

// Returns: (key, iv, path).
func parseArgs() ([]byte, []byte, string, error) {
   var hexKey *string = flag.String("key", "", "the encryption key in hex");
   var hexIV *string = flag.String("iv", "", "the IV in hex");
   var path *string = flag.String("path", "", "the path to the filesystem");
   flag.Parse();

   if (hexKey == nil || *hexKey == "") {
      return nil, nil, "", errors.New("Error: Key required.");
   }

   if (hexIV == nil || *hexIV == "") {
      return nil, nil, "", errors.New("Error: IV required.");
   }

   if (path == nil || *path == "") {
      return nil, nil, "", errors.New("Error: Path required.");
   }

   key, err := hex.DecodeString(*hexKey);
   if (err != nil) {
      return nil, nil, "", errors.Wrap(err, "Could not decode hex key.");
   }

   iv, err := hex.DecodeString(*hexIV);
   if (err != nil) {
      return nil, nil, "", errors.Wrap(err, "Could not decode hex iv.");
   }

   return key, iv, *path, nil;
}

func processCommand(fsDriver *driver.Driver, command string) error {
   args, err := shellquote.Split(command);
   if (err != nil) {
      return errors.Wrap(err, "Failed to split command.");
   }

   var operation string = args[0];
   args = args[1:];

   // Only allow login command if no one is logged in.
   if (activeUser == nil && operation != COMMAND_LOGIN) {
      return errors.New("Need to login.");
   }

   commandFunc, ok := commands[operation];
   if (!ok) {
      return errors.New("Unknown operation: " + operation);
   }

   return errors.Wrap(commandFunc(operation, fsDriver, args), "Failed to run command");
};

func cat(command string, fsDriver *driver.Driver, args []string) error {
   if (len(args) < 1) {
      return errors.New(fmt.Sprintf("USAGE: %s <file> ...", command));
   }

   var buffer []byte = make([]byte, local.IO_BLOCK_SIZE);

   for _, arg := range(args) {
      // Reset the buffer from the last read.
      buffer = buffer[0:cap(buffer)];

      // TODO(eriq): Not root (and root dir)
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

func create(command string, fsDriver *driver.Driver, args []string) error {
   if (len(args) != 2) {
      return errors.New(fmt.Sprintf("USAGE: %s <root email> <root weak passhash>", command));
   }

   return fsDriver.CreateFilesystem(args[0], util.ShaHash(args[1]));
}

func help(command string, fsDriver *driver.Driver, args []string) error {
   return errors.New("Operation not implemented.");
}

func importFile(command string, fsDriver *driver.Driver, args []string) error {
   if (len(args) < 1 || len(args) > 2) {
      return errors.New(fmt.Sprintf("USAGE: %s <external file> [parent id]", command));
   }

   var localPath string = args[0];

   var parent dirent.Id = dirent.ROOT_ID;
   if (len(args) == 2) {
      parent = dirent.Id(args[1]);
   }

   fileReader, err := os.Open(localPath);
   if (err != nil) {
      return errors.Wrap(err, "Failed to open local file for reading: " + localPath);
   }
   defer fileReader.Close();

   // TODO(eriq): Groups Permissions (hard from terminal, just force chmod?).

   // TODO(eriq): Not root
   err = fsDriver.Put(activeUser.Id, filepath.Base(localPath), fileReader, []group.Permission{}, parent);
   if (err != nil) {
      return errors.Wrap(err, "Failed to put imported file: " + localPath);
   }


   return nil;
}

func login(command string, fsDriver *driver.Driver, args []string) error {
   if (len(args) != 2) {
      return errors.New(fmt.Sprintf("USAGE: %s <username> <password>", command));
   }

   authUser, err := fsDriver.UserAuth(args[0], util.ShaHash(args[1]));
   if (err != nil) {
      return errors.Wrap(err, "Failed to authenticate user.");
   }

   activeUser = authUser;
   return nil;
}

func ls(command string, fsDriver *driver.Driver, args []string) error {
   if (len(args) > 1) {
      return errors.New(fmt.Sprintf("USAGE: %s [dir id]", command));
   }

   var id dirent.Id = dirent.ROOT_ID;
   if (len(args) == 1) {
      id = dirent.Id(args[0]);
   }

   entries, err := fsDriver.List(activeUser.Id, id);
   if (err != nil) {
      return errors.Wrap(err, "Failed to list directory: " + string(id));
   }

   for _, entry := range(entries) {
      fmt.Printf("%s\t%s\t%d\t%d\t%s\n", entry.Name, entry.Id, entry.Size, entry.ModTimestamp, entry.Md5);
   }

   return nil;
}

func mkdir(command string, fsDriver *driver.Driver, args []string) error {
   if (len(args) < 1 || len(args) > 2) {
      return errors.New(fmt.Sprintf("USAGE: %s <dir name> [parent id]", command));
   }

   var name string = args[0];

   var parent dirent.Id = dirent.ROOT_ID;
   if (len(args) == 2) {
      parent = dirent.Id(args[1]);
   }

   // TODO(eriq): Not root
   id, err := fsDriver.MakeDir(activeUser.Id, name, parent, []group.Permission{});
   if (err != nil) {
      return errors.Wrap(err, "Failed to make dir: " + name);
   }

   fmt.Println(id);

   return nil;
}
