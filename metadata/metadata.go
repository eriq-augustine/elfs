package metadata;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "encoding/json"

   "github.com/pkg/errors"
)

const (
   // If we have file systems in the wild, we will need to make sure we
   // are looking at consistent structure.
   FORMAT_VERSION = 0
)

// Decode the metadata elements of the metadata file.
// Verify the version and return the size.
func decodeMetadata(decoder *json.Decoder) (int, error) {
   var version int;
   var size int;

   err := decoder.Decode(&version);
   if (err != nil) {
      return 0, errors.Wrap(err, "Could not decode metadata version.");
   }

   if (version != FORMAT_VERSION) {
      return 0, errors.Errorf(
            "Mismatch in FAT version. Expected: %d, Found: %d", FORMAT_VERSION, version);
   }

   err = decoder.Decode(&size);
   if (err != nil) {
      return 0, errors.Wrap(err, "Could not decode metadata size.");
   }

   return size, nil;
}

// Encode the metadata elements of the metadata file.
func encodeMetadata(encoder *json.Encoder, size int) (error) {
   var version int = FORMAT_VERSION;
   err := encoder.Encode(&version);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode metadata version.");
   }

   err = encoder.Encode(&size);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode metadata size.");
   }

   return nil;
}
