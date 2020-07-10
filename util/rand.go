package util;

import (
   "crypto/rand"
   "fmt"
)

const (
   RANDOM_CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func RandomString(length int) string {
   bytes := RandomBytes(length);
   for i, val := range(bytes) {
      bytes[i] = RANDOM_CHARS[int(val) % len(RANDOM_CHARS)];
   }

   return string(bytes)
}

func RandomBytes(length int) []byte {
   if (length <= 0) {
      panic("Number of random bytes must be positive");
   }

   bytes := make([]byte, length);
   _, err := rand.Read(bytes);
   if (err != nil) {
      panic(fmt.Sprintf("Unable to generate random bytes: %v.", err));
   }

   return bytes
}
