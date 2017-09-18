package main;

import (
   "encoding/hex"
   "flag"
   "fmt"

   "github.com/pkg/errors"

   "github.com/eriq-augustine/elfs/util"
)

func main() {
   hexIV, count, err := parseArgs();
   if (err != nil) {
      flag.Usage();
      panic(fmt.Sprintf("Error parsing args: %+v\n", err));
   }

   iv, err := hex.DecodeString(hexIV);
   if (err != nil) {
      panic(fmt.Sprintf("Error decoding IV: %+v\n", err));
   }

   var newIV []byte = append([]byte(nil), iv...);
   util.IncrementBytesByCount(newIV, count);

   fmt.Printf("Raw Hex Key        : [%s]\n", hexIV);
   fmt.Printf("Incremented Hex Key: [%s]\n", hex.EncodeToString(newIV));
}

// Returns: (key, iv, path).
func parseArgs() (string, int, error) {
   var hexIV *string = flag.String("iv", "", "the IV in hex");
   var count *int = flag.Int("count", -1, "the amount to increment the iv");
   flag.Parse();

   if (hexIV == nil || *hexIV == "") {
      return "", 0, errors.New("Error: IV required.");
   }

   if (count == nil || *count < 0) {
      return "", 0, errors.New("Error: Nonnegative path required.");
   }

   return *hexIV, *count, nil;
}
