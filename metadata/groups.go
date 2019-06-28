package metadata;

// Read and write groups from streams.

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/cipherio"
    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/util"
)

// Read all groups into memory.
// This function will not clear the given groups.
// However, the reader WILL be closed.
func ReadGroups(groups map[identity.GroupId]*identity.Group, reader util.ReadSeekCloser) (int, error) {
    version, err := ReadGroupsWithScanner(groups, bufio.NewScanner(reader));
    if (err != nil) {
        return 0, errors.WithStack(err);
    }

    return version, errors.WithStack(reader.Close());
}

// Same as the other read, but we will read directly from a deocder
// owned by someone else.
// This is expecially useful if there are multiple
// sections of metadata written to the same file.
func ReadGroupsWithScanner(groups map[identity.GroupId]*identity.Group, scanner *bufio.Scanner) (int, error) {
    size, version, err := scanMetadata(scanner);
    if (err != nil) {
        return 0, errors.WithStack(err);
    }

    // Read all the groups.
    for i := 0; i < size; i++ {
        var entry identity.Group;

        if (!scanner.Scan()) {
            err = scanner.Err();

            if (err == nil) {
                return 0, errors.Wrapf(io.EOF, "Early end of Groups. Only read %d of %d entries.", i , size);
            } else {
                return 0, errors.Wrapf(err, "Bad scan on Groups entry %d.", i);
            }
        }

        err = json.Unmarshal(scanner.Bytes(), &entry);
        if (err != nil) {
            return 0, errors.Wrapf(err, "Error unmarshaling the group at index %d (%s).", i, string(scanner.Bytes()));
        }

        groups[entry.Id] = &entry;
    }

    return version, nil;
}

// Write all groups.
// This function will not close the given writer.
func WriteGroups(groups map[identity.GroupId]*identity.Group, version int, writer *cipherio.CipherWriter) error {
    err := writeMetadata(writer, len(groups), version);
    if (err != nil) {
        return errors.WithStack(err);
    }

    // Write all the groups.
    for i, entry := range(groups) {
        line, err := json.Marshal(entry);
        if (err != nil) {
            return errors.Wrapf(err, "Failed to marshal Group entry %d.", i);
        }

        _, err = writer.Write([]byte(fmt.Sprintf("%s\n", string(line))));
        if (err != nil) {
            return errors.Wrapf(err, "Failed to write Group entry %d.", i);
        }
    }

    return nil;
}
