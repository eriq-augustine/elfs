package s3;

// A connector that pulls data from an S3 bucket.

import (
   "crypto/cipher"
   "sync"

   "github.com/aws/aws-sdk-go/aws"
   "github.com/aws/aws-sdk-go/aws/credentials"
   "github.com/aws/aws-sdk-go/aws/session"
   "github.com/aws/aws-sdk-go/service/s3"
   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/connector"
   "github.com/eriq-augustine/elfs/dirent"
)

// Keep track of the active connections so two instances don't connect to the same storage.
var activeConnections map[string]bool;
var activeConnectionsLock *sync.Mutex;

func init() {
   activeConnections = make(map[string]bool);
   activeConnectionsLock = &sync.Mutex{};
}

type S3Connector struct {
   bucket string
   s3Client *s3.S3
}

// There should only ever be one connection to a filesystem at a time.
// If an old connection has not been properly closed, then the force parameter
// may be used to cleanup the old connection.
func NewS3Connector(bucket string, credentialsPath string, awsProfile string, region string, force bool) (*S3Connector, error) {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   // TODO(eriq): Lock files?

   _, ok := activeConnections[bucket];
   if (ok) {
      return nil, errors.Errorf("Cannot create two connections to the same storage: %s", bucket);
   }

   var awsCreds *credentials.Credentials = credentials.NewSharedCredentials(credentialsPath, awsProfile);
   // Make sure we can get the credentials.
   _, err := awsCreds.Get();
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   awsSession, err := session.NewSession(&aws.Config{
      Credentials: credentials.NewSharedCredentials(credentialsPath, awsProfile),
      Region: aws.String(region),
   });
   if (err != nil) {
      return nil, errors.Wrap(err, bucket);
   }

   var connector S3Connector = S3Connector {
      bucket: bucket,
      s3Client: s3.New(awsSession),
   };

   err = connector.lock(force);
   if (err != nil) {
      return nil, errors.Wrap(err, bucket);
   }

   return &connector, nil;
}

func (this *S3Connector) GetId() string {
   return connector.CONNECTOR_TYPE_S3 + ":" + this.bucket;
}

func (this *S3Connector) PrepareStorage() error {
   // TODO(eriq)
   return nil;
}

func (this *S3Connector) GetCipherReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (cipherio.ReadSeekCloser, error) {
   return this.getReader(string(fileInfo.Id), blockCipher, fileInfo.IV);
}

func (this *S3Connector) GetMetadataReader(metadataId string, blockCipher cipher.Block, iv []byte) (cipherio.ReadSeekCloser, error) {
   return this.getReader(metadataId, blockCipher, iv);
}

func (this *S3Connector) getReader(id string, blockCipher cipher.Block, iv []byte) (cipherio.ReadSeekCloser, error) {
   ciphertextSize, err := GetSize(this.bucket, id, this.s3Client);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   var reader *S3Reader = NewS3Reader(this.bucket, id, this.s3Client, ciphertextSize);
   return cipherio.NewCipherReader(reader, blockCipher, iv, ciphertextSize);
}

func (this *S3Connector) GetCipherWriter(fileInfo *dirent.Dirent, blockCipher cipher.Block) (*cipherio.CipherWriter, error) {
   // TODO(eriq)
   return nil, nil;
}

func (this *S3Connector) GetMetadataWriter(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherWriter, error) {
   // TODO(eriq)
   return nil, nil;
}

func (this *S3Connector) RemoveFile(file *dirent.Dirent) error {
   // TODO(eriq)
   return nil;
}

func (this *S3Connector) RemoveMetadataFile(metadataId string) error {
   // TODO(eriq)
   return nil;
}

func (this* S3Connector) Close() error {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   activeConnections[this.bucket] = false;
   return errors.WithStack(this.unlock());
}

func (this* S3Connector) lock(force bool) error {
   // TODO(eriq)
   return nil;
}

func (this* S3Connector) unlock() error {
   // TODO(eriq)
   return nil;
}
