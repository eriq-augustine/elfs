package util;

import (
    "math"
)

func MaxInt(a int, b int) int {
    if (a >= b) {
        return a;
    }

    return b;
}

func MinInt(a int, b int) int {
    if (a <= b) {
        return a;
    }

    return b;
}

func MaxInt64(a int64, b int64) int64 {
    if (a >= b) {
        return a;
    }

    return b;
}

func MinInt64(a int64, b int64) int64 {
    if (a <= b) {
        return a;
    }

    return b;
}

func CeilInt(x float32) int {
     return int(math.Ceil(float64(x)))
}

func CeilUint64(x float64) uint64 {
     return uint64(math.Ceil(float64(x)))
}

func CeilInt64(x float64) int64 {
     return int64(math.Ceil(float64(x)))
}
