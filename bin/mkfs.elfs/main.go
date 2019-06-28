package main;

import (
   "fmt"
   "os"

   "github.com/eriq-augustine/elfs/driver"
   "github.com/eriq-augustine/elfs/identity"
   "github.com/eriq-augustine/elfs/util"
)

func main() {
    fsDriver, args := driver.GetDriverFromArgs();
    defer fsDriver.Close();

    if (args.User != identity.ROOT_NAME) {
        fmt.Printf("User must be '%s' for mkfs.", identity.ROOT_NAME);
        os.Exit(1);
    }

    err := fsDriver.CreateFilesystem(util.Weakhash(identity.ROOT_NAME, args.Pass));
    if (err != nil) {
        fmt.Printf("Failed to create filesystem: %+v\n", err);
        os.Exit(2);
    }
}
