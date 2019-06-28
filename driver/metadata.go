package driver;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/dirent"
   "github.com/eriq-augustine/elfs/identity"
   "github.com/eriq-augustine/elfs/metadata"
   "github.com/eriq-augustine/elfs/util"
)

const (
   FAT_ID = "fat"
   USERS_ID = "users"
   GROUPS_ID = "groups"
   SHADOW_SUFFIX = "shadow"

   // Offset the initial IV for each table.
   IV_OFFSET_USERS = 100
   IV_OFFSET_GROUPS = 200
   IV_OFFSET_CACHE = 300
   IV_OFFSET_FAT = 500
)

// Make a copy of the IV and increment it enough.
func (this *Driver) initIVs() {
   this.fatIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.fatIV, IV_OFFSET_FAT);

   this.usersIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.usersIV, IV_OFFSET_USERS);

   this.groupsIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.groupsIV, IV_OFFSET_GROUPS);

   this.cacheIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.cacheIV, IV_OFFSET_CACHE);
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
   version, err := metadata.ReadFat(this.fat, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.fatVersion = version;

   return nil;
}

// Read the full group listing into memory.
func (this *Driver) readGroups() error {
   reader, err := this.connector.GetMetadataReader(GROUPS_ID, this.blockCipher, this.groupsIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.groups = make(map[identity.GroupId]*identity.Group);

   // Metadata takes ownership of reader.
   version, err := metadata.ReadGroups(this.groups, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.groupsVersion = version;

   return nil;
}

// Read the full user listing into memory.
func (this *Driver) readUsers() error {
   reader, err := this.connector.GetMetadataReader(USERS_ID, this.blockCipher, this.usersIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.users = make(map[identity.UserId]*identity.User);

   // Metadata takes ownership of reader.
   version, err := metadata.ReadUsers(this.users, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.usersVersion = version;

   return nil;
}

// Write the full fat to disk.
func (this *Driver) writeFat(shadow bool) error {
   this.fatVersion++;

   var id string = FAT_ID;
   if (shadow) {
      id = id + "_" + SHADOW_SUFFIX;
   }

   err := this.writeFatCore(id, this.fatIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full group listing to disk.
func (this *Driver) writeGroups(shadow bool) error {
   this.groupsVersion++;

   var id string = GROUPS_ID;
   if (shadow) {
      id = id + "_" + SHADOW_SUFFIX;
   }

   err := this.writeGroupsCore(id, this.groupsIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full user listing to disk.
func (this *Driver) writeUsers(shadow bool) error {
   this.usersVersion++;

   var id string = USERS_ID;
   if (shadow) {
      id = id + "_" + SHADOW_SUFFIX;
   }

   err := this.writeUsersCore(id, this.usersIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// The actual FAT write.
func (this *Driver) writeFatCore(metadataId string, iv []byte) error {
   writer, err := this.connector.GetMetadataWriter(metadataId, this.blockCipher, iv);
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = metadata.WriteFat(this.fat, this.fatVersion, writer);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(writer.Close());
}

// The actual groups write.
func (this *Driver) writeGroupsCore(metadataId string, iv []byte) error {
   writer, err := this.connector.GetMetadataWriter(metadataId, this.blockCipher, iv);
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = metadata.WriteGroups(this.groups, this.groupsVersion, writer);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(writer.Close());
}

// The actual users write.
func (this *Driver) writeUsersCore(metadataId string, iv []byte) error {
   writer, err := this.connector.GetMetadataWriter(metadataId, this.blockCipher, iv);
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = metadata.WriteUsers(this.users, this.usersVersion, writer);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return errors.WithStack(writer.Close());
}
