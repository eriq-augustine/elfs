package main;

import (
   "fmt"
   "strings"

   "github.com/eriq-augustine/elfs/driver"
   "github.com/eriq-augustine/elfs/user"
)

// Params: (fs driver, args (not including invocation)).
type commandFunction func(*driver.Driver, *user.User, []string) (error);

type commandArg struct {
   Description string
   Optional bool
}

type commandInfo struct {
   Name string
   Function commandFunction
   Args []commandArg
   Variatic bool
}

func (this commandInfo) ValidateArgs(args []string) bool {
   var minArgs int = 0;
   var maxArgs int = len(this.Args);

   for _, arg := range(this.Args) {
      if (!arg.Optional) {
         minArgs++;
      }
   }

   return len(args) >= minArgs && (this.Variatic || len(args) <= maxArgs);
}

func (this commandInfo) FormatArgs() string {
   if (len(this.Args) == 0) {
      return "";
   }

   var text []string = make([]string, 0, len(this.Args));
   for _, arg := range(this.Args) {
      if (arg.Optional) {
         text = append(text, fmt.Sprintf("[%s]", arg.Description));
      } else {
         text = append(text, fmt.Sprintf("<%s>", arg.Description));
      }
   }

   return strings.Join(text, " ");
}

func (this commandInfo) Usage() string {
   var args string = "";
   if (len(this.Args) > 0) {
      args = " " + this.FormatArgs();
   }

   var variatic string = "";
   if (this.Variatic) {
      variatic = " ...";
   }

   return fmt.Sprintf("   %s%s%s", this.Name, args, variatic);
}
