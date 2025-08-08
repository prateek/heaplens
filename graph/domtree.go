// ABOUTME: Utility functions for working with dominator trees
// ABOUTME: Provides tree traversal and analysis capabilities
package graph

// DominatorDepth computes the depth of each node in the dominator tree.
// Returns a map from node ID to its depth (root has depth 0).
func DominatorDepth(tree map[ObjID][]ObjID) map[ObjID]int {
	depth := make(map[ObjID]int)
	
	// BFS to compute depths
	var computeDepth func(node ObjID, d int)
	computeDepth = func(node ObjID, d int) {
		depth[node] = d
		for _, child := range tree[node] {
			computeDepth(child, d+1)
		}
	}
	
	// Start from super-root
	computeDepth(0, 0)
	
	return depth
}

// DominatorPath returns the path from a node to the root in the dominator tree.
// The path includes the node itself and ends with the root (or super-root).
func DominatorPath(idom map[ObjID]ObjID, node ObjID) []ObjID {
	var path []ObjID
	current := node
	
	// Follow immediate dominators up to root
	for {
		path = append(path, current)
		dom, exists := idom[current]
		if !exists || dom == 0 {
			// Reached root or super-root
			if current != 0 {
				path = append(path, 0) // Add super-root
			}
			break
		}
		current = dom
	}
	
	return path
}

// IsDominated returns true if node is dominated by dominator.
func IsDominated(idom map[ObjID]ObjID, node, dominator ObjID) bool {
	if node == dominator {
		return true // A node dominates itself
	}
	
	current := node
	for {
		dom, exists := idom[current]
		if !exists {
			return false // Reached root without finding dominator
		}
		if dom == dominator {
			return true
		}
		if dom == 0 {
			return dominator == 0 // Only super-root dominates beyond this point
		}
		current = dom
	}
}