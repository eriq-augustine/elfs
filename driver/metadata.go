package driver;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "github.com/eriq-augustine/s3efs/dirent"
   /*
   "encoding/json"
   "fmt"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
   */
)

const (
   // If we have file systems in the wild, we will need to make sure we
   // are looking at consistent structure.
   FORMAT_VERSION = 0
   FAT_ID = "fat"
   USERS_ID = "users"
   GROUPS_ID = "groups"
)

func (this *Driver) readFat() error { return nil; }
func (this *Driver) readUsers() error { return nil; }
func (this *Driver) readGroups() error { return nil; }

/*
// Read the full fat into memory.
func (this *Driver) readFat() error {
   reader, err := this.connector.GetMetadataReader(FAT_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for FAT");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   var metadata metaMetadata;
   err = decoder.Decode(&metadata);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode FAT metadata.");
   }

   if (metadata.Version != FORMAT_VERSION) {
      return errors.WithStack(NewIllegalOperationError(fmt.Sprintf(
            "Mismatch in FAT version. Expected: %d, Found: %d", FORMAT_VERSION, metadata.Version)));
   }

   // Clear the existing fat.
   this.fat = make(map[dirent.Id]*dirent.Dirent);

   // Read all the dirents.
   for {
      var entry dirent.Dirent;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read dirent.");
      }

      this.fat[entry.Id] = &entry;
   }
}

// Read all users into memoty.
func (this *Driver) readUsers() error {
   reader, err := this.connector.GetMetadataReader(USERS_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for users");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   var metadata metaMetadata;
   err = decoder.Decode(&metadata);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode metadata.");
   }

   if (metadata.Version != FORMAT_VERSION) {
      return errors.WithStack(NewIllegalOperationError(fmt.Sprintf(
            "Mismatch in users version. Expected: %d, Found: %d", FORMAT_VERSION, metadata.Version)));
   }

   // Clear the existing users.
   this.users = make(map[user.Id]*user.User);

   // Read all the users.
   for {
      var entry user.User;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read user.");
      }

      this.users[entry.Id] = &entry;
   }
}

// Read all groups into memoty.
func (this *Driver) readGroups() error {
   reader, err := this.connector.GetMetadataReader(GROUPS_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for groups");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   var metadata metaMetadata;
   err = decoder.Decode(&metadata);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode metadata.");
   }

   if (metadata.Version != FORMAT_VERSION) {
      return errors.WithStack(NewIllegalOperationError(fmt.Sprintf(
            "Mismatch in groups version. Expected: %d, Found: %d", FORMAT_VERSION, metadata.Version)));
   }

   // Clear the existing groups.
   this.groups = make(map[group.Id]*group.Group);

   // Read all the groups.
   for {
      var entry group.Group;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read group.");
      }

      this.groups[entry.Id] = &entry;
   }
}

// TODO(eriq): I don't like holding the JSON in memory.
//  I would rather stream it like the read functions.

// Write the full fat to disk.
func (this *Driver) writeFat() error {
   


   reader, err := this.connector.GetMetadataReader(FAT_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for FAT");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   var metadata metaMetadata;
   err = decoder.Decode(&metadata);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode FAT metadata.");
   }

   if (metadata.Version != FORMAT_VERSION) {
      return errors.WithStack(NewIllegalOperationError(fmt.Sprintf(
            "Mismatch in FAT version. Expected: %d, Found: %d", FORMAT_VERSION, metadata.Version)));
   }

   // Clear the existing fat.
   this.fat = make(map[dirent.Id]*dirent.Dirent);

   // Read all the dirents.
   for {
      var entry dirent.Dirent;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read dirent.");
      }

      this.fat[entry.Id] = &entry;
   }
}
*/

// Put this dirent in the semi-durable cache.
func (this *Driver) cacheDirent(direntInfo *dirent.Dirent) {
   // TODO(eriq)
}
