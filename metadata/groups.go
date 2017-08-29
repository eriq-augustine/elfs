package metadata;

// Read and write groups from streams.

import (
   "encoding/json"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/group"
)

// Read all groups into memory.
// This function will not clear the given groups.
// However, the reader WILL be closed.
// We are using a json decoder that may consume extra bytes at the end, therefore
// if we left the reader open it may give inconistent reads.
func ReadGroups(groups map[group.Id]*group.Group, reader cipherio.ReadSeekCloser) (int, error) {
   version, err := ReadGroupsWithDecoder(groups, json.NewDecoder(reader));
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   return version, errors.WithStack(reader.Close());
}

// Same as the other read, but we will read directly from a deocder
// owned by someone else.
// This is expecially useful if there are multiple
// sections of metadata written to the same file
// (since the JSON decoder may read extra bytes).
func ReadGroupsWithDecoder(groups map[group.Id]*group.Group, decoder *json.Decoder) (int, error) {
   size, version, err := decodeMetadata(decoder);
   if (err != nil) {
      return 0, errors.Wrap(err, "Could not decode group metadata.");
   }

   // Read all the groups.
   for i := 0; i < size; i++ {
      var entry group.Group;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return 0, errors.Wrap(err, "Failed to read group.");
      }

      groups[entry.Id] = &entry;
   }

   return version, nil;
}

// Write all groups.
// This function will not close the given reader.
func WriteGroups(groups map[group.Id]*group.Group, version int, writer *cipherio.CipherWriter) error {
   var encoder *json.Encoder = json.NewEncoder(writer);

   err := encodeMetadata(encoder, len(groups), version);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode groups metadata.");
   }

   // Write all the dirents.
   for _, entry := range(groups) {
      err = encoder.Encode(entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to write group.");
      }
   }

   return nil;
}
