package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "bazil.org/fuse"
    "bazil.org/fuse/fs"
    "bazil.org/fuse/fs/fstestutil"
    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/driver"
    "github.com/eriq-augustine/elfs/user"
    "github.com/eriq-augustine/elfs/util"
)

const (
    DEFAULT_MOUNTPOINT = "/tmp/elfs/mount"
)

func main() {
    fsDriver, args := driver.GetDriverFromArgs();
    defer fsDriver.Close();

    // Auth user.
    activeUser, err := fsDriver.UserAuth(args.User, util.Weakhash(args.User, args.Pass));
    if (err != nil) {
        fmt.Printf("Failed to authenticate user: %+v\n", err);
        os.Exit(10);
    }

    // TEST
    fstestutil.DebugByDefault();

    // Mount.
    connection, err := mount(args.Mountpoint);
    if err != nil {
        fmt.Printf("Failed to mount filesystem: %+v\n", err);
        os.Exit(11);
    }

    // Cleanup.
    defer connection.Close();
    defer fuse.Unmount(args.Mountpoint);

    // Try and gracefully handle SIGINT and SIGTERM.
    // Because of how fuse works, we will still need to unmount through umount/fusermount -u.
    sigChan := make(chan os.Signal, 1);
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM);
    go func() {
        <-sigChan;
        connection.Close();
        fuse.Unmount(args.Mountpoint);
        os.Exit(0);
    }();

    // Serve.
    // err = fs.Serve(connection, FS{})
    err = fs.Serve(connection, fuseFS{fsDriver, activeUser})
    if err != nil {
        fmt.Printf("Failed to serve filesystem: %+v\n", err);
        os.Exit(12);
    }

    // Check if the mount process has an error to report.
    <-connection.Ready
    if err := connection.MountError; err != nil {
        fmt.Printf("Error while mounted: %+v\n", err);
        os.Exit(13);
    }
}

func mount(mountpoint string) (*fuse.Conn, error) {
    err := os.MkdirAll(mountpoint, 0700);
    if (err != nil) {
        return nil, err;
    }

    return fuse.Mount(
        mountpoint,

        // Name of the filesystem.
        fuse.FSName("elfs"),
        // Main type is always "fuse".
        fuse.Subtype("elfs"),

        // TODO(eriq): Flag.
        // fuse.ReadOnly(),

        // Prefetch amount in bytes.
        // fuse.MaxReadahead(TODO),

        // TODO
        // fuse.AsyncRead(),
        // fuse.WritebackCache(),
        // fuse.AllowNonEmptyMount(),

        // Allow other users to access the filesystem.
        // fuse.AllowOther(),
        // Mutually exclusive with AllowOther.
        // fuse.AllowRoot(),

        // Allows set-user-identifier or set-group-identifier bits.
        // fuse.AllowSUID(),

        // OSX Only.

        // Local vs network.
        fuse.LocalVolume(),
        // Volume name shown in OSX finder.
        fuse.VolumeName("ELFS"),
        // Disable extended attribute files (e.g. .DS_Store).
        fuse.NoAppleDouble(),
        // Disable extended attributes.
        fuse.NoAppleXattr(),
    );
}

// Implemented interfaces:
//  - fs.FS
type fuseFS struct {
    driver *driver.Driver
    user *user.User
}

func (this fuseFS) Root() (fs.Node, error) {
    fileInfo, err := this.driver.GetDirent(this.user.Id, dirent.ROOT_ID);
    if (err != nil) {
        return nil, errors.Wrap(err, "Unable to get root.");
    }

    return fuseDirent{fileInfo, this.driver, this.user}, nil;
}
