// ABOUTME: Implements Lengauer-Tarjan algorithm for computing dominators in directed graphs
// ABOUTME: Provides O(E α(E,V)) time complexity for finding immediate dominators
package graph

// Dominators computes the immediate dominator for each reachable object in the graph.
// Uses the Lengauer-Tarjan algorithm for O(E α(E,V)) time complexity.
// Returns a map from object ID to its immediate dominator ID.
// The super-root (ID 0) dominates all roots and has no dominator itself.
func Dominators(g Graph) map[ObjID]ObjID {
	// Build adjacency list for forward traversal
	adj := make(map[ObjID][]ObjID)
	allObjects := make([]*Object, 0, g.NumObjects())
	g.ForEachObject(func(obj *Object) {
		allObjects = append(allObjects, obj)
	})
	
	// Add super-root that points to all roots
	roots := g.GetRoots()
	if len(roots.IDs) > 0 {
		adj[0] = roots.IDs // super-root points to all roots
	}
	
	// Build regular edges
	for _, obj := range allObjects {
		if obj.Ptrs != nil {
			adj[obj.ID] = append([]ObjID{}, obj.Ptrs...)
		}
	}
	
	// Run DFS to number nodes and build spanning tree
	var dfsNum int
	vertex := make([]ObjID, 0, len(allObjects)+1)     // DFS number -> vertex ID
	parent := make(map[ObjID]int)                     // vertex -> DFS number of parent in spanning tree
	dfnum := make(map[ObjID]int)                      // vertex -> DFS number
	semi := make(map[ObjID]int)                       // vertex -> DFS number of semidominator
	ancestor := make(map[ObjID]int)                   // for link-eval forest
	idom := make(map[ObjID]ObjID)                     // vertex -> immediate dominator
	samedom := make(map[ObjID]ObjID)                  // for link-eval forest
	best := make(map[ObjID]ObjID)                     // for link-eval forest
	bucket := make(map[int][]ObjID)                   // semidominator -> list of vertices
	
	// DFS from super-root
	var dfs func(v ObjID, p int)
	dfs = func(v ObjID, p int) {
		if _, visited := dfnum[v]; visited {
			return
		}
		
		dfnum[v] = dfsNum
		vertex = append(vertex, v)
		parent[v] = p
		semi[v] = dfsNum
		ancestor[v] = -1
		best[v] = v
		samedom[v] = v
		dfsNum++
		
		for _, w := range adj[v] {
			dfs(w, dfnum[v])
		}
	}
	
	dfs(0, -1) // Start from super-root
	
	// Link-eval functions for path compression
	var compress func(v ObjID)
	compress = func(v ObjID) {
		anc := ancestor[v]
		if anc == -1 {
			return
		}
		ancID := vertex[anc]
		if ancestor[ancID] != -1 {
			compress(ancID)
			if semi[best[ancID]] < semi[best[v]] {
				best[v] = best[ancID]
			}
			ancestor[v] = ancestor[ancID]
		}
	}
	
	eval := func(v ObjID) ObjID {
		if ancestor[v] == -1 {
			return v
		}
		compress(v)
		return best[v]
	}
	
	link := func(v ObjID, w int) {
		ancestor[v] = w
	}
	
	// Process vertices in reverse DFS order
	for i := dfsNum - 1; i > 0; i-- {
		w := vertex[i]
		
		// Step 2: Compute semidominators
		// Consider all predecessors v of w
		for _, v := range allObjects {
			for _, ptr := range v.Ptrs {
				if ptr == w {
					processEdge(v.ID, w, &semi, dfnum, eval, vertex)
				}
			}
		}
		// Also check super-root edges
		for _, ptr := range adj[0] {
			if ptr == w {
				processEdge(0, w, &semi, dfnum, eval, vertex)
			}
		}
		
		// Add w to bucket of its semidominator
		bucket[semi[w]] = append(bucket[semi[w]], w)
		
		// Link w to its parent in the spanning tree
		if parent[w] != -1 {
			link(w, parent[w])
		}
		
		// Step 3: Implicitly compute immediate dominators
		for _, v := range bucket[parent[w]] {
			u := eval(v)
			if semi[u] == semi[v] {
				idom[v] = vertex[parent[w]]
			} else {
				samedom[v] = u
			}
		}
		bucket[parent[w]] = nil
	}
	
	// Step 4: Explicitly compute immediate dominators
	for i := 1; i < dfsNum; i++ {
		w := vertex[i]
		if samedom[w] != w {
			idom[w] = idom[samedom[w]]
		}
	}
	
	// Remove super-root from results
	delete(idom, 0)
	
	return idom
}

func processEdge(v, w ObjID, semi *map[ObjID]int, dfnum map[ObjID]int, eval func(ObjID) ObjID, vertex []ObjID) {
	vNum, vReachable := dfnum[v]
	wNum := dfnum[w]
	
	if !vReachable {
		return // v is not reachable, skip
	}
	
	var u ObjID
	if vNum <= wNum {
		u = v
	} else {
		u = eval(v)
	}
	
	if (*semi)[u] < (*semi)[w] {
		(*semi)[w] = (*semi)[u]
	}
}

// DominatorTree builds a tree structure from immediate dominators.
// Returns a map from each node to its list of immediately dominated nodes.
func DominatorTree(idom map[ObjID]ObjID) map[ObjID][]ObjID {
	tree := make(map[ObjID][]ObjID)
	
	// Initialize with empty slices for all dominators
	for node := range idom {
		tree[node] = []ObjID{}
	}
	tree[0] = []ObjID{} // super-root
	
	// Build tree by reversing idom relationships
	for node, dom := range idom {
		tree[dom] = append(tree[dom], node)
	}
	
	return tree
}