package s3;

// A connector that pulls data from an S3 bucket.

import (
   "crypto/cipher"
   "io/ioutil"
   "os"
   "strings"
   "sync"

   "github.com/aws/aws-sdk-go/aws"
   "github.com/aws/aws-sdk-go/aws/awserr"
   "github.com/aws/aws-sdk-go/aws/credentials"
   "github.com/aws/aws-sdk-go/aws/session"
   "github.com/aws/aws-sdk-go/service/s3"
   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
   "github.com/eriq-augustine/elfs/connector"
   "github.com/eriq-augustine/elfs/dirent"
   "github.com/eriq-augustine/elfs/util"
)

const (
   LOCK_FILENAME = "remote_lock"
   UNKNOWN_HOSTNAME = "unknown"
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
func NewS3Connector(bucket string, credentialsPath string, awsProfile string, region string, endpoint string, force bool) (*S3Connector, error) {
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
      Endpoint: &endpoint,
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

func (this *S3Connector) GetCipherReader(fileInfo *dirent.Dirent, blockCipher cipher.Block) (util.ReadSeekCloser, error) {
   return this.getReader(string(fileInfo.Id), blockCipher, fileInfo.IV);
}

func (this *S3Connector) GetMetadataReader(metadataId string, blockCipher cipher.Block, iv []byte) (util.ReadSeekCloser, error) {
   return this.getReader(metadataId, blockCipher, iv);
}

func (this *S3Connector) getReader(id string, blockCipher cipher.Block, iv []byte) (util.ReadSeekCloser, error) {
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

func (this *S3Connector) checkLock() (string, error) {
   request := &s3.GetObjectInput{
      Bucket: aws.String(this.bucket),
      Key: aws.String(LOCK_FILENAME),
   };

   data, err := this.s3Client.GetObject(request);
   if (err != nil) {
      awsError, ok := err.(awserr.Error);
      if (ok && awsError.Code() == s3.ErrCodeNoSuchKey) {
         return "", nil;
      }

      return "", errors.WithStack(err);
   }

   hostname, err := ioutil.ReadAll(data.Body);
   if (err != nil) {
      return "", errors.WithStack(err);
   }

   return string(hostname), errors.WithStack(data.Body.Close());
}

func (this *S3Connector) writeLock() error {
   hostname, err := os.Hostname();
   if (err != nil) {
      hostname = UNKNOWN_HOSTNAME;
   }

   request := &s3.PutObjectInput{
      Bucket: aws.String(this.bucket),
      Key: aws.String(LOCK_FILENAME),
      Body: strings.NewReader(hostname),
   };

   _, err = this.s3Client.PutObject(request);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}

func (this* S3Connector) lock(force bool) error {
   hostname, err := this.checkLock();
   if (err != nil) {
      return errors.WithStack(err);
   }

   // Lock already exists and we were not told to force it.
   if (hostname != "" && !force) {
      return errors.Errorf("S3 filesystem (at %s) already owned by [%s]." +
            " Ensure that the server is dead and remove the lock or force the connector.",
            this.bucket, hostname);
   }

   // Lock doesn't exist, or we can force it.
   return errors.WithStack(this.writeLock());
}

func (this* S3Connector) unlock() error {
   request := &s3.DeleteObjectInput{
      Bucket: aws.String(this.bucket),
      Key: aws.String(LOCK_FILENAME),
   };

   _, err := this.s3Client.DeleteObject(request);
   if (err != nil) {
      return errors.WithStack(err);
   }

   return nil;
}
