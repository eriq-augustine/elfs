package metadata;

// Read and write users from streams.

import (
   "bufio"
   "encoding/json"
   "fmt"
   "io"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/user"
)

// Read all users into memory.
// This function will not clear the given users.
// However, the reader WILL be closed.
func ReadUsers(users map[user.Id]*user.User, reader cipherio.ReadSeekCloser) (int, error) {
   version, err := ReadUsersWithScanner(users, bufio.NewScanner(reader));
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   return version, errors.WithStack(reader.Close());
}
func ReadUsersWithScanner(users map[user.Id]*user.User, scanner *bufio.Scanner) (int, error) {
   size, version, err := scanMetadata(scanner);
   if (err != nil) {
      return 0, errors.WithStack(err);
   }

   // Read all the users.
   for i := 0; i < size; i++ {
      var entry user.User;

      if (!scanner.Scan()) {
         err = scanner.Err();

         if (err == nil) {
            return 0, errors.Wrapf(io.EOF, "Early end of Users. Only read %d of %d entries.", i , size);
         } else {
            return 0, errors.Wrapf(err, "Bad scan on Users entry %d.", i);
         }
      }

      err = json.Unmarshal(scanner.Bytes(), &entry);
      if (err != nil) {
         return 0, errors.Wrapf(err, "Error unmarshaling the user at index %d (%s).", i, string(scanner.Bytes()));
      }

      users[entry.Id] = &entry;
   }

   return version, nil;
}

// Write all users.
// This function will not close the given writer.
func WriteUsers(users map[user.Id]*user.User, version int, writer *cipherio.CipherWriter) error {
   var bufWriter *bufio.Writer = bufio.NewWriter(writer);

   err := writeMetadata(bufWriter, len(users), version);
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Write all the users.
   for i, entry := range(users) {
      line, err := json.Marshal(entry);
      if (err != nil) {
         return errors.Wrapf(err, "Failed to marshal User entry %d.", i);
      }

      _, err = bufWriter.WriteString(fmt.Sprintf("%s\n", string(line)));
      if (err != nil) {
         return errors.Wrapf(err, "Failed to write User entry %d.", i);
      }
   }

   return errors.WithStack(bufWriter.Flush());
}
