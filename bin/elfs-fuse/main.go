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
    "github.com/spf13/pflag"

    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/driver"
    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/util"
)

const (
    DEFAULT_MOUNTPOINT = "/tmp/elfs/mount"
)

func main() {
    // Add additional command-line options.
    var mountpoint *string = pflag.StringP("mountpoint", "m", DEFAULT_MOUNTPOINT, "The mountpoint of the filesystem.");
    var readonly *bool = pflag.BoolP("readonly", "o", false, "Mount the filesystem as readonly.");
    var debug *bool = pflag.BoolP("debug", "d", false, "Use FUSE debugging.");

    fsDriver, args := driver.GetDriverFromArgs();
    defer fsDriver.Close();

    // Auth user.
    activeUser, err := fsDriver.UserAuth(args.User, util.Weakhash(args.User, args.Pass));
    if (err != nil) {
        fmt.Printf("Failed to authenticate user: %+v\n", err);
        os.Exit(10);
    }

    if (*debug) {
        fstestutil.DebugByDefault();
    }

    // Mount.
    connection, err := mount(*mountpoint, *readonly);
    if err != nil {
        fmt.Printf("Failed to mount filesystem: %+v\n", err);
        os.Exit(11);
    }

    // Cleanup.
    defer connection.Close();
    defer fuse.Unmount(*mountpoint);

    // Try and gracefully handle SIGINT and SIGTERM.
    // Because of how fuse works, we will still need to unmount through umount/fusermount -u.
    sigChan := make(chan os.Signal, 1);
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM);
    go func() {
        <-sigChan;
        connection.Close();
        fuse.Unmount(*mountpoint);
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

func mount(mountpoint string, readonly bool) (*fuse.Conn, error) {
    err := os.MkdirAll(mountpoint, 0700);
    if (err != nil) {
        return nil, err;
    }

    var mountOptions []fuse.MountOption = make([]fuse.MountOption, 0);

    // Name of the filesystem.
    mountOptions = append(mountOptions, fuse.FSName("elfs"));

    // Main type is always "fuse".
    mountOptions = append(mountOptions, fuse.Subtype("elfs"));

    if (readonly) {
        mountOptions = append(mountOptions, fuse.ReadOnly());
    }

    // TODO(eriq): Look into these options.
    // mountOptions = append(mountOptions, fuse.MaxReadahead(ZZZ));
    // mountOptions = append(mountOptions, fuse.AsyncRead());
    // mountOptions = append(mountOptions, fuse.WritebackCache());
    // mountOptions = append(mountOptions, fuse.AllowNonEmptyMount());
    // mountOptions = append(mountOptions, fuse.AllowOther());
    // mountOptions = append(mountOptions, fuse.AllowRoot());
    // mountOptions = append(mountOptions, fuse.AllowSUID());

    // OSX Only.

    // Local vs network.
    mountOptions = append(mountOptions, fuse.LocalVolume());

    // Volume name shown in OSX finder.
    mountOptions = append(mountOptions, fuse.VolumeName("ELFS"));

    // Disable extended attribute files (e.g. .DS_Store).
    mountOptions = append(mountOptions, fuse.NoAppleDouble());

    // Disable extended attributes.
    mountOptions = append(mountOptions, fuse.NoAppleXattr());

    return fuse.Mount(mountpoint, mountOptions...);
}

// Implemented interfaces:
//  - fs.FS
type fuseFS struct {
    driver *driver.Driver
    user *identity.User
}

func (this fuseFS) Root() (fs.Node, error) {
    fileInfo, err := this.driver.GetDirent(this.user.Id, dirent.ROOT_ID);
    if (err != nil) {
        return nil, errors.Wrap(err, "Unable to get root.");
    }

    return fuseDirent{fileInfo, this.driver, this.user}, nil;
}
