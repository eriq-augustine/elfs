package driver;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "os"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/connector/local"
)

func NewLocalDriver(key []byte, iv []byte, path string) (*Driver, error) {
   connector, err := local.NewLocalConnector(path, false);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to get local connector.");
   }

   driver, err := newDriver(key, iv, connector);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   // Try to init the filesystem from any existing metadata.
   err = driver.SyncFromDisk();
   if (err != nil && errors.Cause(err) != nil && !os.IsNotExist(errors.Cause(err))) {
      return nil, errors.WithStack(err);
   }

   return driver, nil;
}
