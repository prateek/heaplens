// ABOUTME: Calculates retained memory sizes using dominator tree analysis
// ABOUTME: Provides efficient computation of memory retained by each object
package graph

// RetainedSize computes the retained size for each reachable object in the graph.
// The retained size of an object is the total size of all objects that would be
// garbage collected if that object were removed. This is computed using the
// dominator tree: an object retains all objects it dominates.
// Returns a map from object ID to its retained size in bytes.
func RetainedSize(g Graph) map[ObjID]uint64 {
	// First compute dominators and build the tree
	dominators := Dominators(g)
	tree := DominatorTree(dominators)
	
	// Create a map to store object sizes
	objSizes := make(map[ObjID]uint64)
	g.ForEachObject(func(obj *Object) {
		objSizes[obj.ID] = obj.Size
	})
	
	// Add super-root with size 0
	objSizes[0] = 0
	
	// Compute retained sizes using post-order traversal of the dominator tree
	retained := make(map[ObjID]uint64)
	
	var computeRetained func(ObjID) uint64
	computeRetained = func(nodeID ObjID) uint64 {
		if size, computed := retained[nodeID]; computed {
			return size
		}
		
		// Start with the object's own size
		size := objSizes[nodeID]
		
		// Add retained sizes of all immediately dominated nodes
		for _, child := range tree[nodeID] {
			size += computeRetained(child)
		}
		
		retained[nodeID] = size
		return size
	}
	
	// Compute retained sizes for all nodes in the tree
	for nodeID := range tree {
		computeRetained(nodeID)
	}
	
	// Remove super-root from results
	delete(retained, 0)
	
	return retained
}

// RetainedSizeSubsets computes retained sizes for a specific subset of objects.
// This is more efficient than computing all retained sizes when you only need
// a few specific objects.
func RetainedSizeSubsets(g Graph, targetIDs []ObjID) map[ObjID]uint64 {
	if len(targetIDs) == 0 {
		return make(map[ObjID]uint64)
	}
	
	// First compute dominators and build the tree
	dominators := Dominators(g)
	tree := DominatorTree(dominators)
	
	// Create a map to store object sizes
	objSizes := make(map[ObjID]uint64)
	g.ForEachObject(func(obj *Object) {
		objSizes[obj.ID] = obj.Size
	})
	
	// Add super-root with size 0
	objSizes[0] = 0
	
	result := make(map[ObjID]uint64)
	computed := make(map[ObjID]uint64) // cache to avoid recomputation
	
	var computeRetained func(ObjID) uint64
	computeRetained = func(nodeID ObjID) uint64 {
		if size, exists := computed[nodeID]; exists {
			return size
		}
		
		// Start with the object's own size
		size := objSizes[nodeID]
		
		// Add retained sizes of all immediately dominated nodes
		for _, child := range tree[nodeID] {
			size += computeRetained(child)
		}
		
		computed[nodeID] = size
		return size
	}
	
	// Compute retained sizes only for target objects that exist
	for _, targetID := range targetIDs {
		if _, exists := objSizes[targetID]; exists && targetID != 0 {
			result[targetID] = computeRetained(targetID)
		}
	}
	
	return result
}