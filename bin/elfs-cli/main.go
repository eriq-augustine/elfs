package main;

import (
   "bufio"
   "encoding/hex"
   "fmt"
   "os"
   "os/signal"
   "strings"
   "syscall"

   "github.com/pkg/errors"
   shellquote "github.com/kballard/go-shellquote"
   "github.com/spf13/pflag"

   "github.com/eriq-augustine/elfs/connector"
   "github.com/eriq-augustine/elfs/driver"
   "github.com/eriq-augustine/elfs/user"
)

const (
   DEFAULT_AWS_CRED_PATH = "config/elfs-wasabi-credentials"
   DEFAULT_AWS_ENDPOINT = ""
   DEFAULT_AWS_PROFILE = "elfsapi"
   DEFAULT_AWS_REGION = "us-east-1"
)

func main() {
   var activeUser *user.User = nil;

   args, err := parseArgs();
   if (err != nil) {
      pflag.Usage();
      fmt.Printf("Error parsing args: %+v\n", err);
      os.Exit(1);
   }

   var fsDriver *driver.Driver = nil;
   if (args.ConnectorType == connector.CONNECTOR_TYPE_LOCAL) {
      fsDriver, err = driver.NewLocalDriver(args.Key, args.IV, args.Path);
      if (err != nil) {
         fmt.Printf("%+v\n", errors.Wrap(err, "Failed to get local driver"));
         os.Exit(2);
      }
   } else if (args.ConnectorType == connector.CONNECTOR_TYPE_S3) {
      fsDriver, err = driver.NewS3Driver(args.Key, args.IV, args.Path, args.AwsCredPath, args.AwsProfile, args.AwsRegion, args.AwsEndpoint);
      if (err != nil) {
         fmt.Printf("%+v\n", errors.Wrap(err, "Failed to get S3 driver"));
         os.Exit(3);
      }
   } else {
      fmt.Printf("Unknown connector type: [%s]\n", args.ConnectorType);
      os.Exit(4);
   }

   // Gracefully handle SIGINT and SIGTERM.
   sigChan := make(chan os.Signal, 1);
   signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM);
   go func() {
      <-sigChan;
      fsDriver.Close();
      os.Exit(0);
   }();

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

func parseArgs() (*args, error) {
   var awsCredPath *string = pflag.StringP("aws-creds", "c", DEFAULT_AWS_CRED_PATH, "Path to AWS credentials");
   var awsEndpoint *string = pflag.StringP("aws-endpoint", "e", DEFAULT_AWS_ENDPOINT, "AWS endpoint to use. Empty string uses standard AWS S3, 'https://s3.wasabisys.com' uses Wasabi, etc..");
   var awsProfile *string = pflag.StringP("aws-profile", "f", DEFAULT_AWS_PROFILE, "AWS profile to use");
   var awsRegion *string = pflag.StringP("aws-region", "r", DEFAULT_AWS_REGION, "AWS region to use");
   var connectorType *string = pflag.StringP("type", "t", "", "Connector type ('s3' or 'local')");
   var hexKey *string = pflag.StringP("key", "k", "", "Encryption key in hex");
   var hexIV *string = pflag.StringP("iv", "i", "", "IV in hex");
   var path *string = pflag.StringP("path", "p", "", "Path to the filesystem");
   pflag.Parse();

   if (hexKey == nil || *hexKey == "") {
      return nil, errors.New("Error: Key required.");
   }

   if (hexIV == nil || *hexIV == "") {
      return nil, errors.New("Error: IV required.");
   }

   if (connectorType == nil || *connectorType == "") {
      // Can't take the address of a constant.
      var tempType string = connector.CONNECTOR_TYPE_LOCAL;
      connectorType = &tempType;
   }

   if (path == nil || *path == "") {
      return nil, errors.New("Error: Path required.");
   }

   key, err := hex.DecodeString(*hexKey);
   if (err != nil) {
      return nil, errors.Wrap(err, "Could not decode hex key.");
   }

   iv, err := hex.DecodeString(*hexIV);
   if (err != nil) {
      return nil, errors.Wrap(err, "Could not decode hex iv.");
   }

   var rtn args = args{
      AwsCredPath: *awsCredPath,
      AwsEndpoint: *awsEndpoint,
      AwsProfile: *awsProfile,
      AwsRegion: *awsRegion,
      ConnectorType: *connectorType,
      Key: key,
      IV: iv,
      Path: *path,
   };

   return &rtn, nil;
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
}

type args struct {
   AwsCredPath string
   AwsEndpoint string
   AwsProfile string
   AwsRegion string
   ConnectorType string
   Key []byte
   IV []byte
   Path string
}
