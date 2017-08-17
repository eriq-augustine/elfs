package metadata;

// Read and write users from streams.

import (
   "encoding/json"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/cipherio"
   "github.com/eriq-augustine/s3efs/user"
)

// Read all users into memory.
// This function will not clear the given users.
// However, the reader WILL be closed.
// We are using a json decoder that may consume extra bytes at the end, therefore
// if we left the reader open it may give inconistent reads.
func ReadUsers(users map[user.Id]*user.User, reader *cipherio.CipherReader) (int, error) {
   version, err := ReadUsersWithDecoder(users, json.NewDecoder(reader));
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   return version, errors.WithStack(reader.Close());
}
func ReadUsersWithDecoder(users map[user.Id]*user.User, decoder *json.Decoder) (int, error) {
   size, version, err := decodeMetadata(decoder);
   if (err != nil) {
      return 0, errors.Wrap(err, "Could not decode user metadata.");
   }

   // Read all the users.
   for i := 0; i < size; i++ {
      var entry user.User;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return 0, errors.Wrap(err, "Failed to read user.");
      }

      users[entry.Id] = &entry;
   }

   return version, nil;
}

// Write all users.
// This function will not close the given reader.
func WriteUsers(users map[user.Id]*user.User, version int, writer *cipherio.CipherWriter) error {
   var encoder *json.Encoder = json.NewEncoder(writer);

   err := encodeMetadata(encoder, len(users), version);
   if (err != nil) {
      return errors.Wrap(err, "Could not encode users metadata.");
   }

   // Write all the dirents.
   for _, entry := range(users) {
      err = encoder.Encode(entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to write user.");
      }
   }

   return nil;
}
