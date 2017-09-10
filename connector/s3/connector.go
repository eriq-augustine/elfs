package s3;

// A connector that pulls data from an S3 bucket.

import (
   "crypto/cipher"
   "strings"
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

const (
   LOCK_KEY = "lock"
   LOCK_TRUE = "true"
   LOCK_FALSE = "false"
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

// Nothing necessary for S3.
func (this *S3Connector) PrepareStorage() error {
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
   return this.getWriter(string(fileInfo.Id), blockCipher, fileInfo.IV, false);
}

func (this *S3Connector) GetMetadataWriter(metadataId string, blockCipher cipher.Block, iv []byte) (*cipherio.CipherWriter, error) {
   return this.getWriter(metadataId, blockCipher, iv, true);
}

func (this *S3Connector) getWriter(id string, blockCipher cipher.Block, iv []byte, isMetadata bool) (*cipherio.CipherWriter, error) {
   writer, err := NewS3Writer(this.bucket, id, this.s3Client, isMetadata);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   return cipherio.NewCipherWriter(writer, blockCipher, iv);
}

func (this *S3Connector) RemoveFile(file *dirent.Dirent) error {
   return errors.WithStack(this.removeFile(string(file.Id)));
}

func (this *S3Connector) RemoveMetadataFile(metadataId string) error {
   return errors.WithStack(this.removeFile(metadataId));
}

func (this *S3Connector) removeFile(id string) error {
   request := &s3.DeleteObjectInput{
      Bucket: aws.String(this.bucket),
      Key: aws.String(id),
   };

   _, err := this.s3Client.DeleteObject(request);
   if (err != nil) {
      return errors.Wrap(err, id);
   }

   return nil;
}

func (this* S3Connector) Close() error {
   activeConnectionsLock.Lock();
   defer activeConnectionsLock.Unlock();

   activeConnections[this.bucket] = false;
   return errors.WithStack(this.unlock());
}

// We lock a bucket by putting a special tag on it.
func (this* S3Connector) lock(force bool) error {
   // Check for an existing lock.
   getRequest := &s3.GetBucketTaggingInput{
      Bucket: aws.String(this.bucket),
   };

   response, err := this.s3Client.GetBucketTagging(getRequest);
   if (err != nil) {
      // Ignore the error that comes up from an empty tag set.
      if (!strings.HasPrefix(err.Error(), "NoSuchTagSet: The TagSet does not exist")) {
         return errors.WithStack(err);
      }
   }

   var isLocked bool = false;
   for _, tag := range(response.TagSet) {
      if (tag.Key != nil && *tag.Key == LOCK_KEY &&
            tag.Value != nil && *tag.Value == LOCK_TRUE) {
         isLocked = true;
      }
   }

   // Lock already exists and we were not told to force it.
   if (isLocked && !force) {
      return errors.Errorf("S3 filesystem (at %s) already owned." +
            " Ensure that no one else is using it or force the connector.",
            this.bucket);
   }

   // Lock doesn't exist, or we can force it.
   putRequest := &s3.PutBucketTaggingInput{
      Bucket: aws.String(this.bucket),
      Tagging: &s3.Tagging{
         TagSet: []*s3.Tag{
            {
               Key: aws.String(LOCK_KEY),
               Value: aws.String(LOCK_TRUE),
            },
         },
      },
   };

   _, err = this.s3Client.PutBucketTagging(putRequest);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

func (this* S3Connector) unlock() error {
   request := &s3.PutBucketTaggingInput{
      Bucket: aws.String(this.bucket),
      Tagging: &s3.Tagging{
         TagSet: []*s3.Tag{
            {
               Key: aws.String(LOCK_KEY),
               Value: aws.String(LOCK_FALSE),
            },
         },
      },
   };

   _, err := this.s3Client.PutBucketTagging(request);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}
