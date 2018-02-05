package metadata;

// Read and write FATs from streams.

import (
   "bufio"
   "encoding/json"
   "fmt"
   "io"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/dirent"
)

// Read a full fat into memory and return the version of
// of the version read.
// This function will not clear the given fat.
// However, the reader WILL be closed.
func ReadFat(fat map[dirent.Id]*dirent.Dirent, reader cipherio.ReadSeekCloser) (int, error) {
   version, err := ReadFatWithScanner(fat, bufio.NewScanner(reader));
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   return version, errors.WithStack(reader.Close());
}

// Same as the other read, but we will read directly from a scanner
// owned by someone else.
// This is expecially useful if there are multiple
// sections of metadata written to the same file.
func ReadFatWithScanner(fat map[dirent.Id]*dirent.Dirent, scanner *bufio.Scanner) (int, error) {
   size, version, err := scanMetadata(scanner);
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   // Read all the dirents.
   for i := 0; i < size; i++ {
      var entry dirent.Dirent;

      if (!scanner.Scan()) {
         err = scanner.Err();

         if (err == nil) {
            return 0, errors.Wrapf(io.EOF, "Early end of FAT. Only read %d of %d entries.", i , size);
         } else {
            return 0, errors.Wrapf(err, "Bad scan on FAT entry %d.", i);
         }
      }

      err = json.Unmarshal(scanner.Bytes(), &entry);
      if (err != nil) {
         return 0, errors.Wrapf(err, "Error unmarshaling the dirent at index %d (%s).", i, string(scanner.Bytes()));
      }

      fat[entry.Id] = &entry;
   }

   return version, nil;
}

// Write a full fat.
// This function will not close the given writer.
func WriteFat(fat map[dirent.Id]*dirent.Dirent, version int, writer *cipherio.CipherWriter) error {
   var bufWriter *bufio.Writer = bufio.NewWriter(writer);

   err := writeMetadata(bufWriter, len(fat), version);
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Write all the dirents.
   for i, entry := range(fat) {
      line, err := json.Marshal(entry);
      if (err != nil) {
         return errors.Wrapf(err, "Failed to marshal FAT entry %d.", i);
      }

      _, err = bufWriter.WriteString(fmt.Sprintf("%s\n", string(line)));
      if (err != nil) {
         return errors.Wrapf(err, "Failed to write FAT entry %d.", i);
      }
   }

   return errors.WithStack(bufWriter.Flush());
}
