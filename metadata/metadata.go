package metadata;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "bufio"
   "fmt"
   "io"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/util"
)

const (
   // If we have file systems in the wild, we will need to make sure we
   // are looking at consistent structure.
   FORMAT_VERSION = 2
)

// Scan the metadata elements of the metadata file.
// Verify the version and return the size and version.
// Note that the version is the metadata version, not the
// format version.
func scanMetadata(scanner *bufio.Scanner) (int, int, error) {
   var formatVersion int;
   var size int;
   var version int;
   var err error;

   formatVersion, err = util.ScanInt(scanner);
   if (err != nil) {
      return 0, 0, errors.WithStack(err);
   }

   if (formatVersion != FORMAT_VERSION) {
      return 0, 0, errors.Errorf(
            "Mismatch in FAT format version. Expected: %d, Found: %d", FORMAT_VERSION, formatVersion);
   }

   size, err = util.ScanInt(scanner);
   if (err != nil) {
      return 0, 0, errors.WithStack(err);
   }

   version, err = util.ScanInt(scanner);
   if (err != nil) {
      return 0, 0, errors.WithStack(err);
   }

   return size, version, nil;
}

// Write the metadata elements of the metadata file.
func writeMetadata(writer io.Writer, size int, version int) (error) {
   _, err := writer.Write([]byte(fmt.Sprintf("%d\n", FORMAT_VERSION)));
   if (err != nil) {
      return errors.WithStack(err);
   }

   _, err = writer.Write([]byte(fmt.Sprintf("%d\n", size)));
   if (err != nil) {
      return errors.WithStack(err);
   }

   _, err = writer.Write([]byte(fmt.Sprintf("%d\n", version)));
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}
