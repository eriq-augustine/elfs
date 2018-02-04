package main;

import (
   "bufio"
   "encoding/hex"
   "fmt"
   "os"
   "strings"

   "github.com/pkg/errors"
   shellquote "github.com/kballard/go-shellquote"
   "github.com/spf13/pflag"

   "github.com/eriq-augustine/elfs/connector"
   "github.com/eriq-augustine/elfs/driver"
   "github.com/eriq-augustine/elfs/user"
)

const (
   AWS_CRED_PATH = "config/elfs-aws-credentials"
   AWS_PROFILE = "elfsapi"
   AWS_REGION = "us-west-2"
)

func main() {
   var activeUser *user.User = nil;

   key, iv, connectorType, path, err := parseArgs();
   if (err != nil) {
      pflag.Usage();
      fmt.Printf("Error parsing args: %+v\n", err);
      os.Exit(1);
   }

   var fsDriver *driver.Driver = nil;
   if (connectorType == connector.CONNECTOR_TYPE_LOCAL) {
      fsDriver, err = driver.NewLocalDriver(key, iv, path);
      if (err != nil) {
         fmt.Printf("%+v\n", errors.Wrap(err, "Failed to get local driver"));
         os.Exit(2);
      }
   } else if (connectorType == connector.CONNECTOR_TYPE_S3) {
      fsDriver, err = driver.NewS3Driver(key, iv, path, AWS_CRED_PATH, AWS_PROFILE, AWS_REGION);
      if (err != nil) {
         fmt.Printf("%+v\n", errors.Wrap(err, "Failed to get S3 driver"));
         os.Exit(3);
      }
   } else {
      fmt.Printf("Unknown connector type: [%s]\n", connectorType);
      os.Exit(4);
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

      err = processCommand(fsDriver, &activeUser, command);
      if (err != nil) {
         fmt.Println("Failed to run command:");
         fmt.Printf("%+v\n", err);
      }
   }
   fmt.Println("");

   fsDriver.Close();
}

// Returns: (key, iv, connector type, path).
func parseArgs() ([]byte, []byte, string, string, error) {
   var hexKey *string = pflag.StringP("key", "k", "", "the encryption key in hex");
   var hexIV *string = pflag.StringP("iv", "i", "", "the IV in hex");
   var connectorType *string = pflag.StringP("type", "t", "", "the connector type ('S3' or 'local')");
   var path *string = pflag.StringP("path", "p", "", "the path to the filesystem");
   pflag.Parse();

   if (hexKey == nil || *hexKey == "") {
      return nil, nil, "", "", errors.New("Error: Key required.");
   }

   if (hexIV == nil || *hexIV == "") {
      return nil, nil, "", "", errors.New("Error: IV required.");
   }

   if (connectorType == nil || *connectorType == "") {
      // Can't take the address of a constant.
      var tempType string = connector.CONNECTOR_TYPE_LOCAL;
      connectorType = &tempType;
   }

   if (path == nil || *path == "") {
      return nil, nil, "", "", errors.New("Error: Path required.");
   }

   key, err := hex.DecodeString(*hexKey);
   if (err != nil) {
      return nil, nil, "", "", errors.Wrap(err, "Could not decode hex key.");
   }

   iv, err := hex.DecodeString(*hexIV);
   if (err != nil) {
      return nil, nil, "", "", errors.Wrap(err, "Could not decode hex iv.");
   }

   return key, iv, *connectorType, *path, nil;
}

func processCommand(fsDriver *driver.Driver, activeUser **user.User, input string) error {
   args, err := shellquote.Split(input);
   if (err != nil) {
      return errors.Wrap(err, "Failed to split command.");
   }

   var command string = args[0];
   args = args[1:];

   commandInfo, ok := commands[command];
   if (!ok) {
      fmt.Printf("Unknown command: [%s].\n", command);
      return nil;
   }

   if (*activeUser == nil && commandInfo.RequireLogin) {
      fmt.Printf("Command [%s] requires login.\n");
      return nil;
   }

   if (!commandInfo.ValidateArgs(args)) {
      fmt.Printf("USAGE: %s\n", commandInfo.Usage());
      return nil;
   }

   result, err := commandInfo.Function(fsDriver, *activeUser, args);
   if (err != nil) {
      return errors.WithStack(err);
   }

   if (commandInfo.Name == COMMAND_LOGIN) {
      *activeUser = result.(*user.User);
   }

   return nil;
};
