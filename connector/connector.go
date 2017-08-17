package connector;

// The connector will handle the operations to the actual backend
// (eg local filesystem or S3).

import (
   "crypto/cipher"

   "github.com/eriq-augustine/s3efs/cipherio"
   "github.com/eriq-augustine/s3efs/dirent"
)

type Connector interface {
   // Every connector should be able to construct a unique id for itself
   // that is the same for each backend.
   GetId() string
   // Prepare the backend storage for initialization.
   PrepareStorage() error
   // Get a reader that transparently handles all decryption.
   GetCipherReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (*cipherio.CipherReader, error)
   // Metadata may be stored in a different way than normal files.
   GetMetadataReader(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherReader, error)
   GetCipherWriter(fileInfo *dirent.Dirent, blockCipher cipher.Block) (*cipherio.CipherWriter, error)
   GetMetadataWriter(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherWriter, error)
   RemoveMetadataFile(metadataId string) error
   RemoveFile(file *dirent.Dirent) error
   Close() error
}
