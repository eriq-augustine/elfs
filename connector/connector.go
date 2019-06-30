package connector;

// The connector will handle the operations to the actual backend
// (eg local filesystem or S3).

import (
   "crypto/cipher"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/dirent"
   "github.com/eriq-augustine/elfs/util"
)

const (
   CONNECTOR_TYPE_LOCAL = "local"
   CONNECTOR_TYPE_S3 = "s3"

   FS_SYS_DIR_ADMIN = "admin"
   FS_SYS_DIR_DATA = "data"
   DATA_GROUP_PREFIX_LEN = 1
)

type Connector interface {
   // Every connector should be able to construct a unique id for itself
   // that is the same for each backend.
   GetId() string
   // Prepare the backend storage for initialization.
   PrepareStorage() error
   // Get a reader that transparently handles all decryption.
   GetCipherReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (util.ReadSeekCloser, error)
   // Metadata may be stored in a different way than normal files.
   GetMetadataReader(metadataId string, blockCipher cipher.Block, iv []byte) (util.ReadSeekCloser, error)
   GetCipherWriter(fileInfo *dirent.Dirent, blockCipher cipher.Block) (*cipherio.CipherWriter, error)
   GetMetadataWriter(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherWriter, error)
   RemoveMetadataFile(metadataId string) error
   RemoveFile(file *dirent.Dirent) error
   Close() error
}
