// ABOUTME: Builds reverse edges for graph traversal
// ABOUTME: Maps objects to their referrers for paths-to-roots

package graph

// ReverseEdges maps each object to the objects that point to it
type ReverseEdges map[ObjID][]ObjID

// BuildReverseEdges creates a map of reverse edges
func BuildReverseEdges(g Graph) ReverseEdges {
	reverse := make(ReverseEdges)
	
	g.ForEachObject(func(obj *Object) {
		for _, targetID := range obj.Ptrs {
			reverse[targetID] = append(reverse[targetID], obj.ID)
		}
	})
	
	return reverse
}