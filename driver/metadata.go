package driver;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "encoding/json"
   "fmt"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

const (
   // If we have file systems in the wild, we will need to make sure we
   // are looking at consistent structure.
   FORMAT_VERSION = 0
   FAT_ID = "fat"
   USERS_ID = "users"
   GROUPS_ID = "groups"
)

// Read the full fat into memory.
func (this *Driver) readFat() error {
   reader, err := this.connector.GetMetadataReader(FAT_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for FAT");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   size, err := decodeMetadata(decoder);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode FAT metadata.");
   }

   // Clear the existing fat.
   this.fat = make(map[dirent.Id]*dirent.Dirent);

   // Read all the dirents.
   for i := 0; i < size; i++ {
      var entry dirent.Dirent;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read dirent.");
      }

      this.fat[entry.Id] = &entry;
   }

   err = reader.Close();
   if (err != nil) {
      return errors.Wrap(err, "Failed to close fat reader.");
   }

   // Build up the directory map.
   this.dirs = dirent.BuildDirs(this.fat);

   return nil;
}

// Write the full fat to disk.
func (this *Driver) writeFat() error {
   writer, err := this.connector.GetMetadataWriter(FAT_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get writer for FAT");
   }

   var encoder *json.Encoder = json.NewEncoder(writer);

   err = encodeMetadata(encoder, len(this.fat));
   if (err != nil) {
      return errors.Wrap(err, "Could not encode FAT metadata.");
   }

   // Write all the dirents.
   for _, entry := range(this.fat) {
      err = encoder.Encode(entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to write dirent.");
      }
   }

   return writer.Close();
}

// Read the full user listing into memory.
func (this *Driver) readUsers() error {
   reader, err := this.connector.GetMetadataReader(USERS_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for users");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   size, err := decodeMetadata(decoder);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode groups metadata.");
   }

   // Clear the existing users.
   this.users = make(map[user.Id]*user.User);

   // Read all the users.
   for i := 0; i < size; i++ {
      var entry user.User;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read user.");
      }

      this.users[entry.Id] = &entry;
   }

   return reader.Close();
}

// Write the full user listing to disk.
func (this *Driver) writeUsers() error {
   writer, err := this.connector.GetMetadataWriter(USERS_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get writer for users");
   }

   var encoder *json.Encoder = json.NewEncoder(writer);

   err = encodeMetadata(encoder, len(this.users));
   if (err != nil) {
      return errors.Wrap(err, "Could not encode users metadata.");
   }

   // Write all the users.
   for _, entry := range(this.users) {
      err = encoder.Encode(entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to write users.");
      }
   }

   return writer.Close();
}

// Read the full group listing into memory.
func (this *Driver) readGroups() error {
   reader, err := this.connector.GetMetadataReader(GROUPS_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get reader for groups");
   }

   var decoder *json.Decoder = json.NewDecoder(reader);

   size, err := decodeMetadata(decoder);
   if (err != nil) {
      return errors.Wrap(err, "Could not decode groups metadata.");
   }

   // Clear the existing groups.
   this.groups = make(map[group.Id]*group.Group);

   // Read all the groups.
   for i := 0; i < size; i++ {
      var entry group.Group;
      err = decoder.Decode(&entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to read group.");
      }

      this.groups[entry.Id] = &entry;
   }

   return reader.Close();
}

// Write the full group listing to disk.
func (this *Driver) writeGroups() error {
   writer, err := this.connector.GetMetadataWriter(GROUPS_ID, this.blockCipher, this.iv);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get writer for groups");
   }

   var encoder *json.Encoder = json.NewEncoder(writer);

   err = encodeMetadata(encoder, len(this.groups));
   if (err != nil) {
      return errors.Wrap(err, "Could not encode groups metadata.");
   }

   // Write all the groups.
   for _, entry := range(this.groups) {
      err = encoder.Encode(entry);
      if (err != nil) {
         return errors.Wrap(err, "Failed to write groups.");
      }
   }

   return writer.Close();
}

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
      return 0, errors.WithStack(NewIllegalOperationError(fmt.Sprintf(
            "Mismatch in FAT version. Expected: %d, Found: %d", FORMAT_VERSION, version)));
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

// Put this dirent in the semi-durable cache.
func (this *Driver) cacheDirent(direntInfo *dirent.Dirent) {
   // TODO(eriq)
}

func (this *Driver) cacheUserAdd(userInfo *user.User) {
   // TODO(eriq)
}

func (this *Driver) cacheUserDel(userInfo *user.User) {
   // TODO(eriq)
}
