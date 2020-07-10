package util;

import (
   "crypto/aes"
   "crypto/cipher"
   "crypto/sha256"
   "encoding/hex"

   "github.com/pkg/errors"
)

const (
   AES_KEY_LENGTH = 32
   DEFAULT_KEY_LENGTH = AES_KEY_LENGTH
   IV_LENGTH = 12  // Pretty standard size (bytes).
)

// Generate a key (random bytes) of the given length (in bytes).
func GenKey(length int) []byte {
   if (length == 0) {
      length = DEFAULT_KEY_LENGTH;
   }

   return RandomBytes(length);
}

func GenAESKey() []byte {
   return RandomBytes(AES_KEY_LENGTH);
}

func GenIV() []byte {
   return RandomBytes(IV_LENGTH);
}

// Get the SHA2-256 string.
func SHA256Hex(val string) string {
   data := sha256.Sum256([]byte(val));
   return hex.EncodeToString(data[:]);
}

// Generate a password hash the same way that clients are expected to.
func Weakhash(username string, password string) string {
   saltedData := username + "." + password + "." + username;
   return SHA256Hex(saltedData);
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
