package driver;

// A driver that just works on a local disk.
// This treats a directory as if it was a partition.

import (
   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/connector/local"
)

func NewLocalDriver(key []byte, iv []byte, path string) (*Driver, error) {
   connector, err := local.NewLocalConnector(path, false);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to get local connector.");
   }

   return newDriver(key, iv, connector);
}
