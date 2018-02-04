package main;

// Just get the weakhash for a username/password so we can use it in testing.

import (
   "fmt"
   "os"

   "github.com/eriq-augustine/elfs/util"
);

func showUsage() {
   fmt.Println("Get a weakhash.");
   fmt.Printf("USAGE: %s <username> <password>\n", os.Args[0]);
}

func main() {
   args := os.Args;

   if (len(args) != 3 || util.SliceHasString(args, "help") || util.SliceHasString(args, "h")) {
      showUsage();
      return;
   }

   fmt.Println(util.Weakhash(args[1], args[2]));
}
