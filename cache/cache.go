package cache;

// The cache acts a a semi-durable write-back cache.
// It will synchroniously write an encrypted set of changes to disk.
// This is to prevent too many full writes of the metadata structures in the backend storage.
// This cache should be checked every time the driver initializes.
// In the case of a crash, the cache may have data that needs to be written to disk.
// All the cached metadata wil be written to the same file: fat, users, and groups.

import (
    "bufio"
    "crypto/cipher"
    "os"
    "path/filepath"
    "sync"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/dirent"
    "github.com/eriq-augustine/elfs/cipherio"
    "github.com/eriq-augustine/elfs/connector"
    "github.com/eriq-augustine/elfs/identity"
    "github.com/eriq-augustine/elfs/metadata"
    "github.com/eriq-augustine/elfs/util"
)

// There should only be one cache for each connector.
var activeCaches map[string]bool;
var activeCachesLock *sync.Mutex;

func init() {
    activeCaches = make(map[string]bool);
    activeCachesLock = &sync.Mutex{};
}

type MetadataCache struct {
    // We will keep the connector id hashed so we don't leak any information.
    connectorId string
    cachePath string
    lock *sync.Mutex
    blockCipher cipher.Block
    iv []byte
    // Nil values represents delete.
    fat map[dirent.Id]*dirent.Dirent
    users map[identity.UserId]*identity.User
    groups map[identity.GroupId]*identity.Group
}

// IV should not have to be transformed to use.
func NewMetadataCache(connector connector.Connector, blockCipher cipher.Block,
        iv []byte) (*MetadataCache, error) {
    activeCachesLock.Lock();
    defer activeCachesLock.Unlock();

    var connectorId string = util.SHA256Hex(connector.GetId());
    _, ok := activeCaches[connectorId];
    if (ok) {
        return nil, errors.New("Cannot create two caches on the same connector.");
    }

    var cachePath string = filepath.Join(os.TempDir(), connectorId);

    var metadataCache *MetadataCache = &MetadataCache{
        connectorId: connectorId,
        cachePath: cachePath,
        lock: &sync.Mutex{},
        blockCipher: blockCipher,
        iv: iv,
        fat: make(map[dirent.Id]*dirent.Dirent),
        users: make(map[identity.UserId]*identity.User),
        groups: make(map[identity.GroupId]*identity.Group),
    };

    err := metadataCache.init();
    if (err != nil) {
        return nil, errors.Wrap(err, "Failed to init cache.");
    }

    return metadataCache, nil;
}

func (this *MetadataCache) Clear() {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.fat =  make(map[dirent.Id]*dirent.Dirent);
    this.users = make(map[identity.UserId]*identity.User);
    this.groups = make(map[identity.GroupId]*identity.Group);

    os.Remove(this.cachePath);
}

func (this *MetadataCache) IsEmpty() bool {
    this.lock.Lock();
    defer this.lock.Unlock();

    return len(this.fat) == 0 && len(this.users) == 0 && len(this.groups) == 0;
}

func (this *MetadataCache) GetFat() map[dirent.Id]*dirent.Dirent {
    this.lock.Lock();
    defer this.lock.Unlock();

    return this.fat;
}

func (this *MetadataCache) GetUsers() map[identity.UserId]*identity.User {
    this.lock.Lock();
    defer this.lock.Unlock();

    return this.users;
}

func (this *MetadataCache) GetGroups() map[identity.GroupId]*identity.Group {
    this.lock.Lock();
    defer this.lock.Unlock();

    return this.groups;
}

// Put this dirent in the semi-durable cache.
func (this *MetadataCache) CacheDirentPut(info *dirent.Dirent) error {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.fat[info.Id] = info;
    return errors.WithStack(this.write());
}

func (this *MetadataCache) CacheDirentDelete(info *dirent.Dirent) error {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.fat[info.Id] = nil;
    return errors.WithStack(this.write());
}

func (this *MetadataCache) CacheUserPut(info *identity.User) error {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.users[info.Id] = info;
    return errors.WithStack(this.write());
}

func (this *MetadataCache) CacheUserDelete(info *identity.User) error {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.users[info.Id] = nil;
    return errors.WithStack(this.write());
}

func (this *MetadataCache) CacheGroupPut(info *identity.Group) error {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.groups[info.Id] = info;
    return errors.WithStack(this.write());
}

func (this *MetadataCache) CacheGroupDelete(info *identity.Group) error {
    this.lock.Lock();
    defer this.lock.Unlock();

    this.groups[info.Id] = nil;
    return errors.WithStack(this.write());
}

func (this *MetadataCache) Close() error {
    this.lock.Lock();
    defer this.lock.Unlock();

    activeCachesLock.Lock();
    defer activeCachesLock.Unlock();

    activeCaches[this.connectorId] = false;
    return nil;
}

func (this *MetadataCache) init() error {
    this.lock.Lock();
    defer this.lock.Unlock();

    return errors.WithStack(this.read());
}

func (this *MetadataCache) read() error {
    file, err := os.Open(this.cachePath);
    if (err != nil) {
        if (os.IsNotExist(err)) {
            return nil;
        }

        return errors.WithStack(err);
    }
    defer file.Close();

    fileStat, err := file.Stat();
    if (err != nil) {
        return errors.WithStack(err);
    }

    reader, err := cipherio.NewCipherReader(file, this.blockCipher, this.iv, fileStat.Size());
    if (err != nil) {
        return errors.WithStack(err);
    }

    var scanner *bufio.Scanner = bufio.NewScanner(reader);

    // Clear the structures before reading.
    this.fat = make(map[dirent.Id]*dirent.Dirent);
    this.users = make(map[identity.UserId]*identity.User);
    this.groups = make(map[identity.GroupId]*identity.Group);

    _, err = metadata.ReadFatWithScanner(this.fat, scanner);
    if (err != nil) {
        return errors.WithStack(err);
    }

    _, err = metadata.ReadUsersWithScanner(this.users, scanner);
    if (err != nil) {
        return errors.WithStack(err);
    }

    _, err = metadata.ReadGroupsWithScanner(this.groups, scanner);
    if (err != nil) {
        return errors.WithStack(err);
    }

    return errors.WithStack(reader.Close());
}

func (this *MetadataCache) write() error {
    file, err := os.Create(this.cachePath);
    if (err != nil) {
        return errors.WithStack(err);
    }
    defer file.Close();

    writer, err := cipherio.NewCipherWriter(file, this.blockCipher, this.iv);
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = metadata.WriteFat(this.fat, 0, writer);
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = metadata.WriteUsers(this.users, 0, writer);
    if (err != nil) {
        return errors.WithStack(err);
    }

    err = metadata.WriteGroups(this.groups, 0, writer);
    if (err != nil) {
        return errors.WithStack(err);
    }

    return errors.WithStack(writer.Close());
}
