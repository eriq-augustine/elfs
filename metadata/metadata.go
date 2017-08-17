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
// Verify the version and return the size and version.
// Note that the version is the metadata version, not the
// format version.
func decodeMetadata(decoder *json.Decoder) (int, int, error) {
   var formatVersion int;
   var size int;
   var version int;

   err := decoder.Decode(&formatVersion);
   if (err != nil) {
      return 0, 0, errors.Wrap(err, "Could not decode metadata format version.");
   }

   if (formatVersion != FORMAT_VERSION) {
      return 0, 0, errors.Errorf(
            "Mismatch in FAT format version. Expected: %d, Found: %d", FORMAT_VERSION, formatVersion);
   }

   err = decoder.Decode(&size);
   if (err != nil) {
      return 0, 0, errors.Wrap(err, "Could not decode metadata size.");
   }

   err = decoder.Decode(&version);
   if (err != nil) {
      return 0, 0, errors.Wrap(err, "Could not decode metadata version.");
   }

   return size, version, nil;
}

// Encode the metadata elements of the metadata file.
func encodeMetadata(encoder *json.Encoder, size int, version int) (error) {
   var formatVersion int = FORMAT_VERSION;
   err := encoder.Encode(&formatVersion);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode metadata format version.");
   }

   err = encoder.Encode(&size);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode metadata size.");
   }

   err = encoder.Encode(&version);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode metadata version.");
   }

   return nil;
}
