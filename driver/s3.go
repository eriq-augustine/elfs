package driver;

import (
   "os"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/connector/s3"
)

func NewS3Driver(key []byte, iv []byte, bucket string, credentialsPath string, awsProfile string, region string) (*Driver, error) {
   connector, err := s3.NewS3Connector(bucket, credentialsPath, awsProfile, region, false);
   if (err != nil) {
      return nil, errors.WithStack(err);
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
