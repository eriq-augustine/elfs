package main;

import (
   "crypto/aes"
   "encoding/hex"
   "flag"
   "fmt"
   "io"
   "os"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/cipherio"
)

func main() {
   key, iv, path, err := parseArgs();
   if (err != nil) {
      flag.Usage();
      panic(fmt.Sprintf("Error parsing args: %+v\n", err));
   }

   blockCipher, err := aes.NewCipher(key)
   if err != nil {
      panic(fmt.Sprintf("Failed to make block cipher: %+v\n", err));
   }

   file, err := os.Open(path);
   if (err != nil) {
      panic(fmt.Sprintf("Failed to open file: %+v\n", err));
   }

   fileStat, err := file.Stat();
   if (err != nil) {
      panic(fmt.Sprintf("Failed to stat file: %+v\n", err));
   }

   reader, err := cipherio.NewCipherReader(file, blockCipher, iv, fileStat.Size());
   if (err != nil) {
      panic(fmt.Sprintf("Failed to create cipher reader: %+v\n", err));
   }
   defer reader.Close();

   _, err = io.Copy(os.Stdout, reader);
   if (err != nil) {
      panic(fmt.Sprintf("Failed to read file: %+v\n", err));
   }
}

// Returns: (key, iv, path).
func parseArgs() ([]byte, []byte, string, error) {
   var hexKey *string = flag.String("key", "", "the encryption key in hex");
   var hexIV *string = flag.String("iv", "", "the IV in hex");
   var path *string = flag.String("path", "", "the path to the ciphertext file");
   flag.Parse();

   if (hexKey == nil || *hexKey == "") {
      return nil, nil, "", errors.New("Error: Key required.");
   }

   if (hexIV == nil || *hexIV == "") {
      return nil, nil, "", errors.New("Error: IV required.");
   }

   if (path == nil || *path == "") {
      return nil, nil, "", errors.New("Error: Path required.");
   }

   key, err := hex.DecodeString(*hexKey);
   if (err != nil) {
      return nil, nil, "", errors.Wrap(err, "Could not decode hex key.");
   }

   iv, err := hex.DecodeString(*hexIV);
   if (err != nil) {
      return nil, nil, "", errors.Wrap(err, "Could not decode hex iv.");
   }

   return key, iv, *path, nil;
}
