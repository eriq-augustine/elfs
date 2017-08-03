package util;

import (
   "crypto/rand"

   "github.com/eriq-augustine/golog"
)

const (
   RANDOM_CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func RandomString(length int) string {
   if (length <= 0) {
      golog.Panic("Random string length must be positive.");
   }

   bytes := make([]byte, length);
   _, err := rand.Read(bytes);
   if (err != nil) {
      golog.PanicE("Unable to generate random string", err);
   }

   for i, val := range(bytes) {
      bytes[i] = RANDOM_CHARS[int(val) % len(RANDOM_CHARS)];
   }

   return string(bytes)
}
