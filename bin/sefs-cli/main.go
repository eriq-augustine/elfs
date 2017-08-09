package main;

import (
   "bufio"
   "encoding/hex"
   "flag"
   "fmt"
   "os"
   "strings"

   "github.com/pkg/errors"
   shellquote "github.com/kballard/go-shellquote"

   "github.com/eriq-augustine/s3efs/driver"
   "github.com/eriq-augustine/s3efs/driver/local"
)

const (
   COMMAND_CAT = "cat"
   COMMAND_CREATE = "create"
   COMMAND_IMPORT = "import"
)

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

   var scanner *bufio.Scanner = bufio.NewScanner(os.Stdin);
   for (scanner.Scan()) {
      var command string = strings.TrimSpace(scanner.Text());

      if (command == "") {
         continue;
      }

      err = processCommand(fsDriver, command);
      if (err != nil) {
         panic(fmt.Sprintf("%+v", errors.Wrap(err, "Failed to run command: " + command)));
      }
   }
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
      case COMMAND_IMPORT:
         return importFile(fsDriver, args);
      default:
         return errors.New("Unknown operation: " + operation);
   }
};

func cat(fsDriver *driver.Driver, args []string) error {
   return errors.New("Operation not implemented.");
}

func create(fsDriver *driver.Driver, args []string) error {
   if (len(args) != 2) {
      return errors.New(fmt.Sprintf("USAGE: %s <root email> <root weak passhash>", COMMAND_CREATE));
   }

   return fsDriver.CreateFilesystem(args[0], args[1]);
}

func importFile(fsDriver *driver.Driver, args []string) error {
   return errors.New("Operation not implemented.");
}
