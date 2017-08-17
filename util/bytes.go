package util;

// "Increment" the byte silce by going through each byte
// (big endian) and incremnt it.
// If the byte does not roll over to zero, then stop there.
func IncrementBytes(bytes []byte) {
   for i, _ := range(bytes) {
      bytes[i]++;

      if (bytes[i] != 0) {
         break;
      }
   }
}

func IncrementBytesByCount(bytes []byte, count int) {
   if (count < 0) {
      panic("Cannot increment bytes by negative count");
   }

   for i := 0; i < count; i++ {
      IncrementBytes(bytes);
   }
}
