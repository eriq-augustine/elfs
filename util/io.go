package util;

import (
   "io"
)

type ReadSeekCloser interface {
   io.Closer
   io.Reader
   io.Seeker
}
