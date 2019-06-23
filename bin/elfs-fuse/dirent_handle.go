package main

// FUSE Handles are lower-level and handle operations on open files (reads/writes).
// The same class (fuseDirent) is used for both nodes and handles.
// This file contains implementations of handle methods.
// Implemented handle interfaces:
//  - fs.HandleFlusher
//  - fs.HandleReadAller
//  - fs.HandleReadDirAller
//  - fs.HandleReader
//  - fs.HandleWriter

import (
    "bytes"
    "fmt"
    "io"
    "syscall"

    "bazil.org/fuse"
    "github.com/pkg/errors"
    "golang.org/x/net/context"
)

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

func (this fuseDirent) ReadAll(ctx context.Context) ([]byte, error) {
    if (!this.dirent.IsFile) {
        return nil, fuse.Errno(syscall.EISDIR);
    }

    var buffer []byte = make([]byte, this.dirent.Size);

    reader, err := this.driver.Read(this.user.Id, this.dirent.Id);
    if (err != nil) {
        return nil, errors.Wrap(err, "Failed to open fs file for reading: " + string(this.dirent.Id));
    }
    defer reader.Close();

    readSize, err := reader.Read(buffer);
    if (err != nil && err != io.EOF) {
        return nil, errors.Wrap(err, "Failed to read fs file: " + string(this.dirent.Id));
    }

    if (uint64(readSize) != this.dirent.Size) {
        return nil, errors.New(fmt.Sprintf("Short read on '%s'. Expected %d, got %d.", this.dirent.Id, this.dirent.Size, readSize));
    }

    return buffer, nil;
}

func (this fuseDirent) Read(ctx context.Context, request *fuse.ReadRequest, response *fuse.ReadResponse) error {
    if (!this.dirent.IsFile) {
        return fuse.Errno(syscall.EISDIR);
    }

    // Ignore all the flags/locks, and just read the contents.
    response.Data = make([]byte, request.Size);

    reader, err := this.driver.Read(this.user.Id, this.dirent.Id);
    if (err != nil) {
        return errors.Wrap(err, "Failed to open fs file for reading: " + string(this.dirent.Id));
    }
    defer reader.Close();

    _, err = reader.Seek(request.Offset, io.SeekStart);
    if (err != nil) {
        return errors.Wrap(err, "Failed to seek for reading: " + string(this.dirent.Id));
    }

    readSize, err := reader.Read(response.Data);
    if (err != nil && err != io.EOF) {
        return errors.Wrap(err, "Failed to read fs file: " + string(this.dirent.Id));
    }

    if (readSize != request.Size) {
        return errors.New(fmt.Sprintf("Short read on '%s'. Expected %d, got %d.", this.dirent.Id, request.Size, readSize));
    }

    return nil
}

func (this fuseDirent) Flush(ctx context.Context, request *fuse.FlushRequest) error {
    // No implementation for Flush is necessary.
    // We won't sync the cache every flush, since that would be pretty expensive.
    return nil;
}

func (this fuseDirent) Write(ctx context.Context, request *fuse.WriteRequest, response *fuse.WriteResponse) error {
    // Although it would be possible to handle writes in the middle of files,
    // we are just keeping it simple and re-writing the entire file.
    // Like Read() we will ignore all flags and locks.

    data, err := this.ReadAll(ctx);
    if (err != nil) {
        return errors.Wrap(err, "Failed to read old data for write: " + string(this.dirent.Id));
    }

    // Just reslice to remove the data after the offset, and append.
    data = data[:request.Offset];
    data = append(data, request.Data...);

    err = this.driver.Put(this.user.Id, this.dirent.Name, bytes.NewReader(data), this.dirent.GroupPermissions, this.dirent.Parent);
    if (err != nil) {
        return errors.Wrap(err, "Failed to write data for write: " + string(this.dirent.Id));
    }

    response.Size = len(request.Data);

    return nil;
}
