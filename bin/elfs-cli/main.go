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
)

func main() {
    var fsDriver *driver.Driver = driver.GetDriverFromArgs();
    defer fsDriver.Close();

    var activeUser *user.User = nil;
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

   if (*activeUser == nil && commandInfo.RequireLogin) {
      fmt.Printf("Command [%s] requires login.\n", command);
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
