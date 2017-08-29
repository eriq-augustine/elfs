package metadata;

// Read and write FATs from streams.

import (
   "encoding/json"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/dirent"
)

// Read a full fat into memory and return the version of
// of the version read.
// This function will not clear the given fat.
// However, the reader WILL be closed.
// We are using a json decoder that may consume extra bytes at the end, therefore
// if we left the reader open it may give inconistent reads.
func ReadFat(fat map[dirent.Id]*dirent.Dirent, reader cipherio.ReadSeekCloser) (int, error) {
   version, err := ReadFatWithDecoder(fat, json.NewDecoder(reader));
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
func ReadFatWithDecoder(fat map[dirent.Id]*dirent.Dirent, decoder *json.Decoder) (int, error) {
   size, version, err := decodeMetadata(decoder);
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   // Read all the dirents.
   for i := 0; i < size; i++ {
      var entry dirent.Dirent;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return 0, errors.WithStack(err);
      }

      fat[entry.Id] = &entry;
   }

   return version, nil;
}

// Write a full fat.
// This function will not close the given reader.
func WriteFat(fat map[dirent.Id]*dirent.Dirent, version int, writer *cipherio.CipherWriter) error {
   var encoder *json.Encoder = json.NewEncoder(writer);

   err := encodeMetadata(encoder, len(fat), version);
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Write all the dirents.
   for _, entry := range(fat) {
      err = encoder.Encode(entry);
      if (err != nil) {
         return errors.WithStack(err);
      }
   }

   return nil;
}
