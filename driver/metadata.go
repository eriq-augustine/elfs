package driver;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/metadata"
   "github.com/eriq-augustine/s3efs/user"
   "github.com/eriq-augustine/s3efs/util"
)

const (
   FAT_ID = "fat"
   USERS_ID = "users"
   GROUPS_ID = "groups"

   // Offset the initial IV for each table.
   IV_OFFSET_USERS = 100
   IV_OFFSET_GROUPS = 200
   IV_OFFSET_CACHE = 300
   IV_OFFSET_FAT = 500
)

// Make a copy of the IV and increment it enough.
func (this *Driver) initIVs() {
   this.fatIV = append([]byte(nil), this.iv...);
   for i := 0; i < IV_OFFSET_FAT; i++ {
      util.IncrementBytes(this.fatIV);
   }

   this.usersIV = append([]byte(nil), this.iv...);
   for i := 0; i < IV_OFFSET_USERS; i++ {
      util.IncrementBytes(this.usersIV);
   }

   this.groupsIV = append([]byte(nil), this.iv...);
   for i := 0; i < IV_OFFSET_GROUPS; i++ {
      util.IncrementBytes(this.groupsIV);
   }

   this.cacheIV = append([]byte(nil), this.iv...);
   for i := 0; i < IV_OFFSET_CACHE; i++ {
      util.IncrementBytes(this.cacheIV);
   }
}

// Read the full fat into memory.
func (this *Driver) readFat() error {
   reader, err := this.connector.GetMetadataReader(FAT_ID, this.blockCipher, this.fatIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Clear the existing fat.
   this.fat = make(map[dirent.Id]*dirent.Dirent);

   // Metadata takes ownership of reader.
   err = metadata.ReadFat(this.fat, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full fat to disk.
func (this *Driver) writeFat() error {
   writer, err := this.connector.GetMetadataWriter(FAT_ID, this.blockCipher, this.fatIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = metadata.WriteFat(this.fat, writer);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(writer.Close());
}

// Read the full user listing into memory.
func (this *Driver) readUsers() error {
   reader, err := this.connector.GetMetadataReader(USERS_ID, this.blockCipher, this.usersIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.users = make(map[user.Id]*user.User);

   // Metadata takes ownership of reader.
   err = metadata.ReadUsers(this.users, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full user listing to disk.
func (this *Driver) writeUsers() error {
   writer, err := this.connector.GetMetadataWriter(USERS_ID, this.blockCipher, this.usersIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = metadata.WriteUsers(this.users, writer);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(writer.Close());
}

// Read the full group listing into memory.
func (this *Driver) readGroups() error {
   reader, err := this.connector.GetMetadataReader(GROUPS_ID, this.blockCipher, this.groupsIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.groups = make(map[group.Id]*group.Group);

   // Metadata takes ownership of reader.
   err = metadata.ReadGroups(this.groups, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full group listing to disk.
func (this *Driver) writeGroups() error {
   writer, err := this.connector.GetMetadataWriter(GROUPS_ID, this.blockCipher, this.groupsIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = metadata.WriteGroups(this.groups, writer);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(writer.Close());
}
