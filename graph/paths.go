// ABOUTME: BFS algorithm for finding paths from objects to GC roots
// ABOUTME: Implements K-shortest paths with cycle detection

package graph

// Path represents a path from an object to a root
type Path struct {
	IDs []ObjID // Sequence of object IDs from target to root
}

// PathsToRoots finds paths from an object to GC roots using BFS
func PathsToRoots(g Graph, from ObjID, maxPaths int) []Path {
	if maxPaths <= 0 {
		return nil
	}
	
	// Build reverse edges for traversal
	reverse := BuildReverseEdges(g)
	
	// Get roots
	roots := g.GetRoots()
	rootSet := make(map[ObjID]bool)
	for _, id := range roots.IDs {
		rootSet[id] = true
	}
	
	// Check if starting object is itself a root
	if rootSet[from] {
		return []Path{{IDs: []ObjID{from}}}
	}
	
	// BFS state
	type searchNode struct {
		id   ObjID
		path []ObjID
	}
	
	var result []Path
	queue := []searchNode{{id: from, path: []ObjID{from}}}
	
	// BFS to find paths
	for len(queue) > 0 && len(result) < maxPaths {
		node := queue[0]
		queue = queue[1:]
		
		// Get objects that point to current node
		referrers := reverse[node.id]
		
		for _, referrerID := range referrers {
			// Avoid cycles by checking if we've already visited this node in this path
			inPath := false
			for _, id := range node.path {
				if id == referrerID {
					inPath = true
					break
				}
			}
			if inPath {
				continue
			}
			
			newPath := make([]ObjID, len(node.path)+1)
			copy(newPath, node.path)
			newPath[len(node.path)] = referrerID
			
			// Check if we reached a root
			if rootSet[referrerID] {
				result = append(result, Path{IDs: newPath})
				if len(result) >= maxPaths {
					break
				}
			} else {
				// Continue searching
				queue = append(queue, searchNode{
					id:   referrerID,
					path: newPath,
				})
			}
		}
	}
	
	return result
}