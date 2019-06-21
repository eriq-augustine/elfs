package main;

import (
   "bufio"
   "fmt"
   "os"
   "strings"

   "github.com/pkg/errors"
   shellquote "github.com/kballard/go-shellquote"

   "github.com/eriq-augustine/elfs/driver"
   "github.com/eriq-augustine/elfs/user"
   "github.com/eriq-augustine/elfs/util"
)

func main() {
    fsDriver, args := driver.GetDriverFromArgs();
    defer fsDriver.Close();

    activeUser, err := fsDriver.UserAuth(args.User, util.Weakhash(args.User, args.Pass));
    if (err != nil) {
        fmt.Printf("Failed to authenticate user: %+v\n", err);
        os.Exit(2);
    }

    var scanner *bufio.Scanner = bufio.NewScanner(os.Stdin);

    for {
        fmt.Printf("%s > ", activeUser.Name);

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

        err := processCommand(fsDriver, &activeUser, command);
        if (err != nil) {
            fmt.Println("Failed to run command:");
            fmt.Printf("%+v\n", err);
        }
    }
    fmt.Println("");
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

   if (!commandInfo.ValidateArgs(args)) {
      fmt.Printf("USAGE: %s\n", commandInfo.Usage());
      return nil;
   }

   err = commandInfo.Function(fsDriver, *activeUser, args);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}
