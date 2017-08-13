package metadata;

// Read and write FATs from streams.

import (
   "encoding/json"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/cipherio"
   "github.com/eriq-augustine/s3efs/dirent"
)

// Read a full fat into memory.
// This function will not clear the given fat.
// However, the reader WILL be closed.
// We are using a json decoder that may consume extra bytes at the end, therefore
// if we left the reader open it may give inconistent reads.
func ReadFat(fat map[dirent.Id]*dirent.Dirent, reader *cipherio.CipherReader) error {
   err := ReadFatWithDecoder(fat, json.NewDecoder(reader));
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(reader.Close());
}

// Same as the other read, but we will read directly from a deocder
// owned by someone else.
// This is expecially useful if there are multiple
// sections of metadata written to the same file
// (since the JSON decoder may read extra bytes).
func ReadFatWithDecoder(fat map[dirent.Id]*dirent.Dirent, decoder *json.Decoder) error {
   size, err := decodeMetadata(decoder);
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Read all the dirents.
   for i := 0; i < size; i++ {
      var entry dirent.Dirent;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.WithStack(err);
      }

      fat[entry.Id] = &entry;
   }

   return nil;
}

// Write a full fat.
// This function will not close the given reader.
func WriteFat(fat map[dirent.Id]*dirent.Dirent, writer *cipherio.CipherWriter) error {
   var encoder *json.Encoder = json.NewEncoder(writer);

   err := encodeMetadata(encoder, len(fat));
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
