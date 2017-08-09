package main;

import (
   "encoding/hex"
   "fmt"

   "github.com/eriq-augustine/s3efs/util"
)

func main() {
   fmt.Println("Key: " + hex.EncodeToString(util.GenAESKey()));
   fmt.Println("IV : " + hex.EncodeToString(util.GenIV()));
}
