package util;

// Some utilities for slices.

// Uses standard n search and equality.
func SliceHasString(haystack []string, needle string) bool {
   for _, val := range(haystack) {
      if (val == needle) {
         return true;
      }
   }

   return false;
}
