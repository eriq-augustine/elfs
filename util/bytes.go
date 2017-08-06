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
