package util;

import (
   "crypto/rand"
   "crypto/sha256"
   "fmt"

   "github.com/eriq-augustine/golog"
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
