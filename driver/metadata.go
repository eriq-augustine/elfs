package driver;

// Helpers that deal only with metadata (fat, users, and groups).

import (
   "fmt"

   "github.com/eriq-augustine/golog"
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

   // Offsets for base IVs for the shadow tables.
   IV_OFFSET_SHADOW_USERS = 1100
   IV_OFFSET_SHADOW_GROUPS = 1200
   IV_OFFSET_SHADOW_FAT = 1500

   // The number of shadow tables to keep around.
   SHADOW_COUNT = 10
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

   this.shadowFatIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.shadowFatIV, IV_OFFSET_SHADOW_FAT);

   this.shadowUsersIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.shadowUsersIV, IV_OFFSET_SHADOW_USERS);

   this.shadowGroupsIV = append([]byte(nil), this.iv...);
   util.IncrementBytesByCount(this.shadowGroupsIV, IV_OFFSET_SHADOW_GROUPS);
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

   this.groups = make(map[group.Id]*group.Group);

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

   this.users = make(map[user.Id]*user.User);

   // Metadata takes ownership of reader.
   version, err := metadata.ReadUsers(this.users, reader);
   if (err != nil) {
      return errors.WithStack(err);
   }

   this.usersVersion = version;

   return nil;
}

// Write the full fat to disk.
func (this *Driver) writeFat() error {
   this.fatVersion++;

   err := this.writeShadowFat();
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = this.writeFatCore(FAT_ID, this.fatIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full group listing to disk.
func (this *Driver) writeGroups() error {
   this.groupsVersion++;

   err := this.writeShadowGroups();
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = this.writeGroupsCore(GROUPS_ID, this.groupsIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// Write the full user listing to disk.
func (this *Driver) writeUsers() error {
   this.usersVersion++;

   err := this.writeShadowUsers();
   if (err != nil) {
      return errors.WithStack(err);
   }

   err = this.writeUsersCore(USERS_ID, this.usersIV);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

// More internal write functions that handle shadow tables.

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

// Shadow writes.

// Write the shadow fat and remove any shadow that is too old.
func (this *Driver) writeShadowFat() error {
   var iv []byte = append([]byte(nil), this.shadowFatIV...);
   util.IncrementBytesByCount(iv, this.fatVersion);

   // Ignore failed remove of old shadow.
   if (this.fatVersion > SHADOW_COUNT) {
      err := this.connector.RemoveMetadataFile(getShadowId(FAT_ID, this.fatVersion - SHADOW_COUNT));
      if (err != nil) {
         golog.WarnE("Failed to remove shadow fat.", err);
      }
   }

   return errors.WithStack(this.writeFatCore(getShadowId(FAT_ID, this.fatVersion), iv));
}

// Write the shadow groups and remove any shadow that is too old.
func (this *Driver) writeShadowGroups() error {
   var iv []byte = append([]byte(nil), this.shadowGroupsIV...);
   util.IncrementBytesByCount(iv, this.groupsVersion);

   // Ignore failed remove of old shadow.
   if (this.groupsVersion > SHADOW_COUNT) {
      err := this.connector.RemoveMetadataFile(getShadowId(GROUPS_ID, this.groupsVersion - SHADOW_COUNT));
      if (err != nil) {
         golog.WarnE("Failed to remove shadow groups.", err);
      }
   }

   return errors.WithStack(this.writeGroupsCore(getShadowId(GROUPS_ID, this.groupsVersion), iv));
}

// Write the shadow users and remove any shadow that is too old.
func (this *Driver) writeShadowUsers() error {
   var iv []byte = append([]byte(nil), this.shadowUsersIV...);
   util.IncrementBytesByCount(iv, this.usersVersion);

   // Ignore failed remove of old shadow.
   if (this.usersVersion > SHADOW_COUNT) {
      err := this.connector.RemoveMetadataFile(getShadowId(USERS_ID, this.usersVersion - SHADOW_COUNT));
      if (err != nil) {
         golog.WarnE("Failed to remove shadow users.", err);
      }
   }

   return errors.WithStack(this.writeUsersCore(getShadowId(USERS_ID, this.usersVersion), iv));
}

func getShadowId(metadataId string, version int) string {
   return fmt.Sprintf("%s_shadow_%06d", metadataId, version);
}
