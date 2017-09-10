package s3;

// A readseeker that connects to S3.
// Note that unlike cipherio.CipherReader, S3Reader does not have to
// worry at all about the size of the cleartext.
// That will all be handled in cipherio.CipherReader.

// TODO(eriq): Right now every read will triger a request, we should fetch ahead and cache.

import (
   "fmt"
   "io"
   "syscall"

   "github.com/aws/aws-sdk-go/aws"
   "github.com/aws/aws-sdk-go/service/s3"
   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/util"
)

type S3Reader struct {
   bucket *string
   objectId *string
   s3Client *s3.S3
   offset int64
   ciphertextSize int64
}

func NewS3Reader(bucket string, objectId string, s3Client *s3.S3, ciphertextSize int64) *S3Reader {
   return &S3Reader{
      bucket: aws.String(bucket),
      objectId: aws.String(objectId),
      s3Client: s3Client,
      offset: 0,
      ciphertextSize: ciphertextSize,
   };
}

func (this *S3Reader) Read(outBuffer []byte) (int, error) {
   // Return EOF if we are already at the end.
   if (this.offset >= this.ciphertextSize) {
      return 0, io.EOF;
   }

   // TEST
   fmt.Printf("Read Request: %d bytes\n", len(outBuffer));

   // Figure out the end for this read.
   var requestEndOffset int64 = util.MinInt64(this.ciphertextSize, this.offset + int64(len(outBuffer)));
   var byteRange string = fmt.Sprintf("bytes=%d-%d", this.offset, requestEndOffset);

   // TEST
   fmt.Printf("   Byte Range: [%s]\n", byteRange);

   request := &s3.GetObjectInput{
      Bucket: this.bucket,
      Key: this.objectId,
      Range: aws.String(byteRange),
   };

   object, err := this.s3Client.GetObject(request);
   if (err != nil) {
      return 0, errors.Wrap(err, *this.objectId);
   }

   if (object.ContentLength == nil) {
      return 0, errors.Errorf("Got a nil content length: %s", *this.objectId);
   }

   // TEST
   fmt.Printf("   Content Length: %d\n", *object.ContentLength);

   // Resize the output buffer to fit what was actaully sent.
   outBuffer = outBuffer[0:*object.ContentLength];

   // TODO(eriq): Verify that we got the right size
   readSize, err := io.ReadFull(object.Body, outBuffer);
   this.offset += int64(readSize);
   if (err != nil && err != io.EOF) {
      return readSize, errors.WithStack(err);
   }

   // TEST
   fmt.Printf("   Read size: %d\n", readSize);

   err = object.Body.Close();
   if (err != nil) {
      return readSize, errors.WithStack(err);
   }

   // Return EOF if we have read to the end.
   err = nil;
   if (this.offset >= this.ciphertextSize) {
      err = io.EOF;
   }

   return readSize, err;
}

func (this *S3Reader) Seek(offset int64, whence int) (int64, error) {
   absoluteOffset, err := this.absoluteSeekOffset(offset, whence);
   if (err != nil) {
      return this.offset, errors.WithStack(err);
   }

   // It is not strange to Seek(io.SeekCurrent, 0) just to see where the reader is.
   if (absoluteOffset == this.offset) {
      return this.offset, nil;
   }

   // Change the cleartext offset.
   this.offset = absoluteOffset;

   return this.offset, nil;
}

// Deall with all the different wences and give the absolute offset from the start of the file.
// If the seek offset is not valid in any way, a corresponding error will be retutned.
func (this *S3Reader) absoluteSeekOffset(offset int64, whence int) (int64, error) {
   switch whence {
      case io.SeekStart:
         // Nothing to do.
      case io.SeekCurrent:
         offset = this.offset + offset;
      case io.SeekEnd:
         offset = this.ciphertextSize + offset;
      default:
         return 0, errors.Wrapf(syscall.EINVAL, "Unknown whence for seek: %d", whence);
   }

   if (offset < 0 || offset > this.ciphertextSize) {
      return 0, errors.WithStack(syscall.EINVAL);
   }

   return offset, nil;
}

func (this *S3Reader) Close() error {
   return nil;
}

// Utility that just gets an object's size.
// Will require one fetch.
func GetSize(bucket string, objectId string, s3Client *s3.S3) (int64, error) {
   request := &s3.HeadObjectInput{
      Bucket: aws.String(bucket),
      Key: aws.String(objectId),
   };

   response, err := s3Client.HeadObject(request);
   if (err != nil) {
      return 0, errors.Wrap(err, objectId);
   }

   if (response.ContentLength == nil) {
      return 0, errors.Errorf("Content length does not exist on response: %s", objectId);
   }

   return *response.ContentLength, nil;
}
