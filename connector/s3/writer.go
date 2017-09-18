package s3;

import (
   "bytes"

   "github.com/aws/aws-sdk-go/aws"
   "github.com/aws/aws-sdk-go/service/s3"
   "github.com/pkg/errors"
)

type S3Writer struct {
   bucket *string
   objectId *string
   s3Client *s3.S3
   // We need to keep the identifiers for each part for when we complete the multiplart upload.
   parts []*s3.CompletedPart
   // We must be extra cautious about making sure this is closed on success or failure.
   // Not closing could result in billable storage being used by intermitent data.
   uploadId *string
}

func NewS3Writer(bucket string, objectId string, s3Client *s3.S3, isMetadata bool) (*S3Writer, error) {
   var storageClass string = s3.StorageClassStandardIa;
   if (isMetadata) {
      storageClass = s3.StorageClassStandard;
   }

   request := &s3.CreateMultipartUploadInput{
      Bucket: aws.String(bucket),
      Key: aws.String(objectId),
      StorageClass: aws.String(storageClass),
   };

   response, err := s3Client.CreateMultipartUpload(request);
   if (err != nil) {
      return nil, errors.Wrap(err, objectId);
   }

   if (response.UploadId == nil) {
      return nil, errors.Errorf("Could not get upload id for: [%s]", objectId);
   }
   var uploadId string = *response.UploadId;

   return &S3Writer{
      bucket: aws.String(bucket),
      objectId: aws.String(objectId),
      s3Client: s3Client,
      parts: make([]*s3.CompletedPart, 0),
      uploadId: aws.String(uploadId),
   }, nil;
}

func (this *S3Writer) Write(data []byte) (int, error) {
   // Return EOF if we are already at the end.
   if (len(data) == 0 || this.uploadId == nil) {
      return 0, nil;
   }

   request := &s3.UploadPartInput{
      Bucket: this.bucket,
      Key: this.objectId,
      Body: bytes.NewReader(data),
      PartNumber: aws.Int64(int64(len(this.parts) + 1)),  // One-indexed.
      UploadId: this.uploadId,
   };

   response, err := this.s3Client.UploadPart(request);
   if (err != nil) {
      this.Abort();
      return 0, errors.Wrap(err, *this.objectId);
   }

   if (response.ETag == nil) {
      this.Abort();
      return 0, errors.Errorf("Reponse does not have an ETag: [%s]", *this.objectId);
   }
   var etag string = *response.ETag;

   this.parts = append(this.parts, &s3.CompletedPart{
      PartNumber: aws.Int64(int64(len(this.parts) + 1)),  // One-indexed
      ETag: aws.String(etag),
   });

   return len(data), nil;
}

func (this *S3Writer) Abort() error {
   if (this.uploadId == nil) {
      return nil;
   }

   request := &s3.AbortMultipartUploadInput{
      Bucket: this.bucket,
      Key: this.objectId,
      UploadId: this.uploadId,
   };

   this.uploadId = nil;

   _, err := this.s3Client.AbortMultipartUpload(request);
   return errors.WithStack(err);
}

func (this *S3Writer) Close() error {
   if (this.uploadId == nil) {
      return nil;
   }

   request := &s3.CompleteMultipartUploadInput{
      Bucket: this.bucket,
      Key: this.objectId,
      UploadId: this.uploadId,
      MultipartUpload: &s3.CompletedMultipartUpload{Parts: this.parts},
   };

   this.uploadId = nil;
   this.parts = nil;

   _, err := this.s3Client.CompleteMultipartUpload(request);
   return errors.WithStack(err);
}
