package dirent;

// Maintain a list of directories and their children.

const (
   FILE_SEPARATOR = "/"
)

// Take in the FULL fat and create a mapping of directories to their children.
// This is not a tree, just a straight up map.
// Only directories will be keys and every directory will be represented.
func BuildDirs(fat map[Id]*Dirent) (map[Id][]*Dirent) {
   var dirs map[Id][]*Dirent = make(map[Id][]*Dirent);

   for _, dirent := range(fat) {
      // Dirs will first make sure that they are represented.
      if (!dirent.IsFile) {
         _, ok := dirs[dirent.Id];
         if (!ok) {
            dirs[dirent.Id] = make([]*Dirent, 0);
         }
      }

      // Now dirs and files alike will ensure that their parent exists
      // and then put themselves in their parent's children.

      // Skip root.
      if (dirent.Id == ROOT_ID) {
         continue;
      }

      _, ok := dirs[dirent.Parent];
      if (!ok) {
         dirs[dirent.Parent] = make([]*Dirent, 0, 1);
      }

      dirs[dirent.Parent] = append(dirs[dirent.Parent], dirent);
   }

   return dirs;
}
