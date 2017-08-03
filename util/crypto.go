package util;

import (
   "crypto/rand"

   "github.com/eriq-augustine/golog"
)

const (
   AES_KEY_LENGTH = 32
   DEFAULT_KEY_LENGTH = AES_KEY_LENGTH
)

// Generate a key (random bytes) of the given length (in bytes).
func GenKey(length int) []byte {
   if (length == 0) {
      length = DEFAULT_KEY_LENGTH;
   }

   bytes := make([]byte, length);
   _, err := rand.Read(bytes);
   if (err != nil) {
      golog.PanicE("Unable to generate random key", err);
   }

   return bytes
}
