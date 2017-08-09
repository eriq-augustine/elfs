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

const (
   COMMAND_CAT = "cat"
   COMMAND_CREATE = "create"
   COMMAND_HELP = "help"
   COMMAND_IMPORT = "import"
   COMMAND_LOGIN = "login"
   COMMAND_LS = "ls"
   COMMAND_QUIT = "quit"
)

// TODO(eriq): login
// TODO(eriq): hash pass

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
      fmt.Print("> ");
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
         panic(fmt.Sprintf("%+v", errors.Wrap(err, "Failed to run command: " + command)));
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

   switch operation {
      case COMMAND_CAT:
         return cat(fsDriver, args);
      case COMMAND_CREATE:
         return create(fsDriver, args);
      case COMMAND_HELP:
         return help(fsDriver, args);
      case COMMAND_IMPORT:
         return importFile(fsDriver, args);
      case COMMAND_LOGIN:
         return login(fsDriver, args);
      case COMMAND_LS:
         return ls(fsDriver, args);
      default:
         return errors.New("Unknown operation: " + operation);
   }
};

func cat(fsDriver *driver.Driver, args []string) error {
   if (len(args) < 1) {
      return errors.New(fmt.Sprintf("USAGE: %s <file> ...", COMMAND_CREATE));
   }

   // TODO(eriq): Don't take raw ids, take paths.

   var buffer []byte = make([]byte, local.IO_BLOCK_SIZE);

   for _, arg := range(args) {
      // Reset the buffer from the last read.
      buffer = buffer[0:cap(buffer)];

      // TODO(eriq): Not root (and root dir)
      reader, err := fsDriver.Read(user.ROOT_ID, dirent.Id(arg));
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

func create(fsDriver *driver.Driver, args []string) error {
   if (len(args) != 2) {
      return errors.New(fmt.Sprintf("USAGE: %s <root email> <root weak passhash>", COMMAND_CREATE));
   }

   return fsDriver.CreateFilesystem(args[0], util.ShaHash(args[1]));
}

func help(fsDriver *driver.Driver, args []string) error {
   return errors.New("Operation not implemented.");
}

func importFile(fsDriver *driver.Driver, args []string) error {
   if (len(args) < 1) {
      return errors.New(fmt.Sprintf("USAGE: %s <file> ...", COMMAND_CREATE));
   }

   // TODO(eriq): dest path

   for _, arg := range(args) {
      fileReader, err := os.Open(arg);
      if (err != nil) {
         return errors.Wrap(err, "Failed to open local file for reading: " + arg);
      }

      // TODO(eriq): Not root (and root dir)
      err = fsDriver.Put(user.ROOT_ID, filepath.Base(arg), fileReader, []group.Permission{}, dirent.ROOT_ID);
      if (err != nil) {
         return errors.Wrap(err, "Failed to put imported file: " + arg);
      }

      fileReader.Close();
   }

   return nil;
}

func login(fsDriver *driver.Driver, args []string) error {
   return errors.New("Operation not implemented.");
}

func ls(fsDriver *driver.Driver, args []string) error {
   // TODO(eriq): context path?
   return errors.New("Operation not implemented.");
}
