package util;

import (
   "crypto/aes"
   "crypto/cipher"
   "crypto/rand"
   "crypto/sha256"
   "fmt"

   "github.com/eriq-augustine/golog"
   "github.com/pkg/errors"
)

const (
   AES_KEY_LENGTH = 32
   DEFAULT_KEY_LENGTH = AES_KEY_LENGTH
   IV_LENGTH = 12  // Pretty standard size (bytes).
)

// Generate some crypto random bytes.
func GenBytes(length int) []byte {
   if (length <= 0) {
      golog.Panic("Number of random bytes must be positive");
   }

   bytes := make([]byte, length);
   _, err := rand.Read(bytes);
   if (err != nil) {
      golog.PanicE("Unable to generate random bytes", err);
   }

   return bytes
}

// Generate a key (random bytes) of the given length (in bytes).
func GenKey(length int) []byte {
   if (length == 0) {
      length = DEFAULT_KEY_LENGTH;
   }

   return GenBytes(length);
}

func GenAESKey() []byte {
   return GenBytes(AES_KEY_LENGTH);
}

func GenIV() []byte {
   return GenBytes(IV_LENGTH);
}

// Hash a string and get back the hex string.
func ShaHash(data string) string {
   return fmt.Sprintf("%x", sha256.Sum256([]byte(data)));
}

// One-off encryption and decryption.
// This is not meant for huge chunks of data.
func Encrypt(key []byte, iv []byte, cleartext []byte) ([]byte, error) {
   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      return nil, errors.WithStack(err);
   }

   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, errors.WithStack(err);
   }

   var ciphertext []byte = make([]byte, 0, len(cleartext) + gcm.Overhead());

   ciphertext = gcm.Seal(ciphertext, iv, cleartext, nil);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   return ciphertext, nil;
}

func Decrypt(key []byte, iv []byte, ciphertext []byte) ([]byte, error) {
   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      return nil, errors.WithStack(err);
   }

   gcm, err := cipher.NewGCM(blockCipher);
   if err != nil {
      return nil, errors.WithStack(err);
   }

   var cleartext []byte = make([]byte, 0, len(ciphertext) - gcm.Overhead());

   cleartext, err = gcm.Open(cleartext, iv, ciphertext, nil);
   if (err != nil) {
      return nil, errors.WithStack(err);
   }

   return cleartext, nil;
}
