package driver;

// Core filesystem operations that do not operate on single files.

import (
   "time"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/s3efs/dirent"
   "github.com/eriq-augustine/s3efs/group"
   "github.com/eriq-augustine/s3efs/user"
)

func (this *Driver) Close() {
   this.SyncToDisk();
   this.connector.Close();
}

// Create a new filesystem.
func (this *Driver) CreateFilesystem(rootPasshash string) error {
   this.connector.PrepareStorage();

   rootUser, err := user.New(user.ROOT_ID, rootPasshash, user.ROOT_NAME);
   if (err != nil) {
      return errors.Wrap(err, "Could not create root user.");
   }

   this.users[rootUser.Id] = rootUser;

   this.groups[group.EVERYBODY_ID] = group.New(group.EVERYBODY_ID, group.EVERYBODY_NAME, rootUser.Id);

   var permissions []group.Permission = []group.Permission{group.NewPermission(group.EVERYBODY_ID, true, true)};
   this.fat[dirent.ROOT_ID] = dirent.NewDir(dirent.ROOT_ID, rootUser.Id, dirent.ROOT_NAME,
         permissions, dirent.ROOT_ID, time.Now().Unix());

   // Force a write of the FAT, users, and groups.
   this.SyncToDisk();

   return nil;
}

// Read all the metadata from disk into memory.
// This should only be done once when the driver initializes.
func (this *Driver) SyncFromDisk() error {
   err := this.readFat();
   if (err != nil) {
      return errors.Wrap(err, "Could not read FAT");
   }

   err = this.readUsers();
   if (err != nil) {
      return errors.Wrap(err, "Could not read users");
   }

   err = this.readGroups();
   if (err != nil) {
      return errors.Wrap(err, "Could not read groups");
   }

   // Also check the cache for incomplete transactions.
   err = this.loadFromCache();
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Build up the directory map.
   this.dirs = dirent.BuildDirs(this.fat);

   return nil;
}

// Write all metadata to disk and clear the cache after.
func (this *Driver) SyncToDisk() error {
   err := this.writeFat();
   if (err != nil) {
      return errors.Wrap(err, "Could not write FAT");
   }

   err = this.writeUsers();
   if (err != nil) {
      return errors.Wrap(err, "Could not write users");
   }

   err = this.writeGroups();
   if (err != nil) {
      return errors.Wrap(err, "Could not write groups");
   }

   // All changes are on disk, the cache is safe to clear.
   this.cache.Clear();

   return nil;
}

// Read the cache and if there are entries, sync them to disk.
// Nil values in the cache represents deletes.
func (this *Driver) loadFromCache() error {
   if (this.cache.IsEmpty()) {
      return nil;
   }

   for id, entry := range(this.cache.GetFat()) {
      if (entry == nil) {
         delete(this.fat, id);
      } else {
         this.fat[id] = entry;
      }
   }

   for id, entry := range(this.cache.GetUsers()) {
      if (entry == nil) {
         delete(this.users, id);
      } else {
         this.users[id] = entry;
      }
   }

   for id, entry := range(this.cache.GetGroups()) {
      if (entry == nil) {
         delete(this.groups, id);
      } else {
         this.groups[id] = entry;
      }
   }

   return errors.WithStack(this.SyncToDisk());
}
