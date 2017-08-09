package dirent;

// A node in our directory structure.

import (
   "fmt"
   "strings"

   "github.com/pkg/errors"
)

const (
   FILE_SEPARATOR = "/"
)

// Anything that can be in a directory.
type Node struct {
   Id Id
   IsFile bool
   // Map each child by name to their node.
   Children map[string]*Node
}

// Add a node to this tree.
// The context node better contain the dirent.
func (this *Node) AddNode(fat map[Id]*Dirent, dirent *Dirent) error {
   // Get the parent (path then node).
   parentPath, err := GetPath(fat, dirent.Parent);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get parent path for " + string(dirent.Id));
   }

   parent, err := this.GetNode(parentPath);
   if (err != nil) {
      return errors.Wrap(err, "Failed to get parent node for " + string(dirent.Id));
   }

   parent.Children[dirent.Name] = &Node{
      Id: dirent.Id,
      IsFile: dirent.IsFile,
      Children: make(map[string]*Node),
   };

   return nil;
}

func (this *Node) GetNode(path string) (*Node, error) {
   node, err := this.getNode(strings.Split(path, FILE_SEPARATOR));
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to get node for path: " + path);
   }

   return node, nil;
}

func (this *Node) getNode(path []string) (*Node, error) {
   if (len(path) == 0) {
      return this, nil;
   }

   child, ok := this.Children[path[0]];
   if (!ok) {
      return nil, errors.Errorf("Could not find child by name: %s", path[0]);
   }

   return child.getNode(path[1:]);
}

// Construct the absolute path for a dirent.
func GetPath(fat map[Id]*Dirent, id Id) (string, error) {
   path, err := getPath(fat, id, nil);
   if (err != nil) {
      return "", errors.Wrap(err, "Failed to construct path");
   }

   return strings.Join(path, FILE_SEPARATOR), nil;
}

func getPath(fat map[Id]*Dirent, id Id, path []string) ([]string, error) {
   dirent, ok := fat[id];
   if (!ok) {
      return nil, errors.Errorf("Unknown dirent (%s) while trying to generate path.", id);
   }

   // Prepend this dirent, then move to the parent.
   path = append([]string{dirent.Name}, path...);

   if (dirent.Parent != ROOT_ID) {
      // Go is giving me a strange error below where if I use := it says that path
      // is declared and not used.
      var err error;
      path, err = getPath(fat, dirent.Parent, path);
      if (err != nil) {
         return nil, errors.Errorf("Error getting path for parent of %s", id);
      }
   }

   return path, nil;
}

// Construct a tree for an entire FAT.
// The FAT better be complete.
func BuildTree(fat map[Id]*Dirent) (*Node, error) {
   var visited map[Id]*Node = make(map[Id]*Node);

   // Preload with root.
   visited[ROOT_ID] = &Node{
      Id: ROOT_ID,
      IsFile: false,
      Children: make(map[string]*Node),
   }

   for _, dirent := range(fat) {
      _, err := visitNode(visited, fat, dirent);
      if (err != nil) {
         return nil, errors.Wrap(err, "Failed to visit dirent: " + string(dirent.Id));
      }
   }

   return visited[ROOT_ID], nil;
}

func visitNode(visited map[Id]*Node, fat map[Id]*Dirent, dirent *Dirent) (*Node, error) {
   var node *Node;

   node, ok := visited[dirent.Id];
   if (ok) {
      return node, nil;
   }

   // Visit the parent.
   parent, ok := fat[dirent.Parent];
   if (!ok) {
      return nil, errors.New(fmt.Sprintf("Could not locate parent (%s) for child (%s)", dirent.Parent, dirent.Id));
   }

   parentNode, err := visitNode(visited, fat, parent);
   if (err != nil) {
      return nil, errors.Wrap(err, "Failed to visit parent: " + string(parent.Id));
   }

   node = &Node{
      Id: dirent.Id,
      IsFile: dirent.IsFile,
      Children: make(map[string]*Node),
   };

   // Put the child in the parent's children.
   parentNode.Children[dirent.Name] = node;

   return node, nil;
}
