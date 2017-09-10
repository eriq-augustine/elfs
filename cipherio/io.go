package cipherio;

const (
   // When doing reads or writes, the size of data to work with in bytes.
   // 5MB is the minimum size for an aws multipart upload.
   IO_BLOCK_SIZE = 1024 * 1024 * 5
)
