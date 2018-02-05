package util;

import (
   "bufio"
   "io"
   "strconv"

   "github.com/pkg/errors"
)

// Will Scan() the scanner once and read the contents as a string.
func ScanInt(scanner *bufio.Scanner) (int, error) {
   if (!scanner.Scan()) {
      return 0, io.EOF;
   }

   val, err := strconv.Atoi(string(scanner.Bytes()));
   if (err != nil) {
      return 0, errors.Wrapf(err, "Failed of scan int on '%s'.", string(scanner.Bytes()));
   }

   return val, nil;
}
