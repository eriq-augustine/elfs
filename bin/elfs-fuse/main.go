package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "bazil.org/fuse"
    "bazil.org/fuse/fs"
    _ "bazil.org/fuse/fs/fstestutil"
    "github.com/pkg/errors"
    "golang.org/x/net/context"

    "github.com/eriq-augustine/elfs/cipherio"
    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/driver"
    "github.com/eriq-augustine/elfs/user"
    "github.com/eriq-augustine/elfs/util"
)

const (
    DEFAULT_MOUNTPOINT = "/tmp/elfs/mount"
    FUSE_BLOCKSIZE = 512
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

        fuse.ReadOnly(),

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

// Implemented interfaces:
//  - fs.Node
//  - fs.NodeStringLookuper
//  - fs.HandleReadDirAller
type fuseDirent struct {
    dirent *dirent.Dirent
    driver *driver.Driver
    user *user.User
}

func (this fuseDirent) Attr(ctx context.Context, attr *fuse.Attr) error {
    attr.Inode = 0;  // Dynamic.
    attr.Size = this.dirent.Size;
    attr.Blocks = util.CeilUint64(float64(this.dirent.Size) / FUSE_BLOCKSIZE);
    attr.Atime = time.Unix(this.dirent.AccessTimestamp, 0);
    attr.Mtime = time.Unix(this.dirent.ModTimestamp, 0);
    attr.Ctime = time.Unix(this.dirent.CreateTimestamp, 0);
    attr.Crtime = time.Unix(this.dirent.CreateTimestamp, 0);
    attr.Nlink = 1;
    attr.Uid = uint32(this.dirent.Owner);
    attr.Gid = 0;  // Group permissions are more of an ACL.
    // attr.Rdev
    // attr.Flags
    attr.BlockSize = cipherio.IO_BLOCK_SIZE;

    if (this.dirent.IsFile) {
        attr.Mode = 0555;
    } else {
        attr.Mode = os.ModeDir | 0555;
    }

    return nil;
}

func (this fuseDirent) Lookup(ctx context.Context, name string) (fs.Node, error) {
    if (this.dirent.IsFile) {
        return nil, fuse.ENOENT;
    }

    // Get the children for this dir.
    entries, err := this.driver.List(this.user.Id, this.dirent.Id);
    if (err != nil) {
        return nil, errors.Wrap(err, "Failed to list directory: " + string(this.dirent.Id));
    }

    for _, entry := range(entries) {
        if (entry.Name != name) {
            continue;
        }

        return fuseDirent{entry, this.driver, this.user}, nil;
    }

    return nil, fuse.ENOENT;
}

func (this fuseDirent) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
    if (this.dirent.IsFile) {
        return nil, fuse.ENOENT;
    }

    // Get the children for this dir.
    entries, err := this.driver.List(this.user.Id, this.dirent.Id);
    if (err != nil) {
        return nil, errors.Wrap(err, "Failed to list directory: " + string(this.dirent.Id));
    }

    var rtn []fuse.Dirent = make([]fuse.Dirent, 0, len(entries));

    for _, entry := range(entries) {
        var direntType fuse.DirentType = fuse.DT_Dir;
        if (entry.IsFile) {
            direntType = fuse.DT_File;
        }

        var fuseDirent fuse.Dirent = fuse.Dirent{
            Inode: 0,
            Type: direntType,
            Name: entry.Name,
        };

        rtn = append(rtn, fuseDirent);
    }

    return rtn, nil;
}
