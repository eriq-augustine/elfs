package driver;

// Core filesystem operations that do not operate on single files.

import (
    "time"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/identity"
)

func (this *Driver) Close() {
    this.SyncToDisk(false);
    this.connector.Close();
}

func (this *Driver) ConnectionString() string {
    return this.connector.GetId();
}

// Create a new filesystem.
func (this *Driver) CreateFilesystem(rootPasshash string) error {
    this.connector.PrepareStorage();

    rootUser, rootGroup, err := identity.NewUser(identity.ROOT_USER_ID, identity.ROOT_NAME, rootPasshash, identity.ROOT_GROUP_ID);
    if (err != nil) {
        return errors.Wrap(err, "Could not create root user.");
    }

    this.users[rootUser.Id] = rootUser;
    this.groups[rootGroup.Id] = rootGroup;

    this.cache.CacheUserPut(rootUser);
    this.cache.CacheGroupPut(rootGroup);

    this.fat[dirent.ROOT_ID] = dirent.NewDir(dirent.ROOT_ID, dirent.ROOT_NAME, dirent.ROOT_ID,
            rootUser.Id, rootGroup.Id, time.Now().Unix());

    // Force a write of the FAT, users, and groups.
    this.SyncToDisk(true);

    return nil;
}

// Read all the metadata from disk into memory.
// This should only be done once when the driver initializes.
func (this *Driver) SyncFromDisk() error {
    err := this.readMetadata();
    if (err != nil) {
        return errors.WithStack(err);
    }

    // If the metadata has been successfully read, write it back out to a shadow.
    err = this.writeMetadata(true);
    if (err != nil) {
        return errors.WithStack(err);
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
func (this *Driver) SyncToDisk(force bool) error {
    if (!force && this.cache.IsEmpty()) {
        return nil;
    }

    err := this.writeMetadata(false);
    if (err != nil) {
        return errors.WithStack(err);
    }

    // All changes are on disk, the cache is safe to clear.
    this.cache.Clear();

    return nil;
}

func (this *Driver) readMetadata() error {
    err := this.readFat();
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = this.readUsers();
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = this.readGroups();
    if (err != nil) {
        return errors.WithStack(err);
    }

    return nil;
}

func (this *Driver) writeMetadata(shadow bool) error {
    err := this.writeFat(shadow);
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = this.writeUsers(shadow);
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = this.writeGroups(shadow);
    if (err != nil) {
        return errors.WithStack(err);
    }

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

    return errors.WithStack(this.SyncToDisk(false));
}
