package incr

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

// newRecomputeHeap returns a new recompute heap with a given maximum height.
func newRecomputeHeap(maxHeight int) *recomputeHeap {
	return &recomputeHeap{
		heights: make([]map[Identifier]INode, maxHeight),
		lookup:  make(map[Identifier]INode),
	}
}

// recomputeHeap is a height ordered list of lists of nodes.
type recomputeHeap struct {
	// mu synchronizes critical sections for the heap.
	mu sync.Mutex

	// minHeight is the smallest heights index that has nodes
	minHeight int
	// maxHeight is the largest heights index that has nodes
	maxHeight int

	// heights is an array of linked lists corresponding
	// to node heights. it should be pre-allocated with
	// the constructor to the height limit number of elements.
	heights []map[Identifier]INode
	// lookup is a quick lookup function for testing if an item exists
	// in the heap, and specifically removing single elements quickly by id.
	lookup map[Identifier]INode
}

// clear completely resets the recompute heap, preserving
// its current capacity.
func (rh *recomputeHeap) clear() {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.heights = make([]map[Identifier]INode, len(rh.heights))
	clear(rh.lookup)
	rh.minHeight = 0
	rh.maxHeight = 0
}

func (rh *recomputeHeap) len() int {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	return len(rh.lookup)
}

func (rh *recomputeHeap) add(nodes ...INode) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.addUnsafe(nodes...)
}

func (rh *recomputeHeap) fix(ids ...Identifier) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.fixUnsafe(ids...)
}

func (rh *recomputeHeap) has(s INode) (ok bool) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	_, ok = rh.lookup[s.Node().id]
	return
}

// removeMin removes the minimum height node.
func (rh *recomputeHeap) removeMin() (node INode, ok bool) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	node, ok = rh.removeMinUnsafe()
	return
}

// removeMin removes the minimum height node.
func (rh *recomputeHeap) removeMinUnsafe() (node INode, ok bool) {
	for x := rh.minHeight; x <= rh.maxHeight; x++ {
		if rh.heights[x] != nil && len(rh.heights[x]) > 0 {
			node, ok = popMap(rh.heights[x])
			node.Node().heightInRecomputeHeap = heightUnset
			delete(rh.lookup, node.Node().id)
			if len(rh.heights[x]) > 0 {
				rh.minHeight = x
			} else {
				rh.minHeight = rh.nextMinHeightUnsafe()
			}
			return
		}
	}
	return
}

// removeMinHeight removes the minimum height nodes from
// the recompute heap all at once.
func (rh *recomputeHeap) removeMinHeight() (nodes []INode) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	if rh.heights[rh.minHeight] != nil && len(rh.heights[rh.minHeight]) > 0 {
		nodes = make([]INode, 0, len(rh.heights[rh.minHeight]))
		for id, n := range rh.heights[rh.minHeight] {
			n.Node().heightInRecomputeHeap = heightUnset
			nodes = append(nodes, n)
			delete(rh.lookup, id)
		}
		clear(rh.heights[rh.minHeight])
		rh.minHeight = rh.nextMinHeightUnsafe()
	}
	return
}

func (rh *recomputeHeap) remove(s INode) (ok bool) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	sn := s.Node()
	var item INode
	item, ok = rh.lookup[sn.id]
	if !ok {
		return
	}

	rh.removeItemUnsafe(item)
	return
}

//
// utils
//

func (rh *recomputeHeap) fixUnsafe(ids ...Identifier) {
	for _, id := range ids {
		if item, ok := rh.lookup[id]; ok {
			delete(rh.heights[item.Node().heightInRecomputeHeap], id)
			delete(rh.lookup, id)
			item.Node().heightInRecomputeHeap = heightUnset
			rh.addNodeUnsafe(item)
		}
	}
}

func (rh *recomputeHeap) addUnsafe(nodes ...INode) {
	for _, s := range nodes {
		rh.addNodeUnsafe(s)
	}
}

func (rh *recomputeHeap) addNodeUnsafe(s INode) {
	sn := s.Node()
	height := sn.height
	s.Node().heightInRecomputeHeap = height
	rh.maybeUpdateMinMaxHeights(height)
	rh.maybeAddNewHeights(height)
	if rh.heights[height] == nil {
		rh.heights[height] = make(map[Identifier]INode)
	}
	rh.heights[height][sn.id] = s
	rh.lookup[sn.id] = s
}

func (rh *recomputeHeap) removeItemUnsafe(item INode) {
	id := item.Node().id
	height := item.Node().heightInRecomputeHeap
	delete(rh.lookup, id)
	delete(rh.heights[height], id)

	// handle the edge case where removing a node removes the _last_ node
	// in the current minimum height list, causing us to need to move
	// the minimum height up one value.
	isLastAtHeight := rh.heights[height] == nil || len(rh.heights[height]) == 0
	if height == rh.minHeight && isLastAtHeight {
		rh.minHeight = rh.nextMinHeightUnsafe()
	}
	item.Node().heightInRecomputeHeap = heightUnset
}

func (rh *recomputeHeap) maybeUpdateMinMaxHeights(newHeight int) {
	if len(rh.lookup) == 0 {
		rh.minHeight = newHeight
		rh.maxHeight = newHeight
		return
	}
	if rh.minHeight > newHeight {
		rh.minHeight = newHeight
	}
	if rh.maxHeight < newHeight {
		rh.maxHeight = newHeight
	}
}

func (rh *recomputeHeap) maybeAddNewHeights(newHeight int) {
	if len(rh.heights) <= newHeight {
		required := (newHeight - len(rh.heights)) + 1
		for x := 0; x < required; x++ {
			rh.heights = append(rh.heights, nil)
		}
	}
}

// nextMinHeightUnsafe finds the next smallest height in the heap that has nodes.
func (rh *recomputeHeap) nextMinHeightUnsafe() (next int) {
	if len(rh.lookup) == 0 {
		return
	}
	for x := rh.minHeight; x <= rh.maxHeight; x++ {
		if len(rh.heights[x]) > 0 {
			next = x
			break
		}
	}
	return
}

// sanityCheck loops through each item in each height block
// and checks that all the height values match.
func (rh *recomputeHeap) sanityCheck() error {
	if len(rh.lookup) > 0 && len(rh.heights[rh.minHeight]) == 0 {
		return fmt.Errorf("recompute heap; sanity check; lookup has items but min height block is empty")
	}
	for heightIndex, height := range rh.heights {
		if height == nil {
			continue
		}
		for _, item := range height {
			if item.Node().heightInRecomputeHeap != heightIndex {
				return fmt.Errorf("recompute heap; sanity check; at height %d item has height %d", heightIndex, item.Node().heightInRecomputeHeap)
			}
			if item.Node().heightInRecomputeHeap != item.Node().height {
				return fmt.Errorf("recompute heap; sanity check; at height %d item has height %d and node has height %d", heightIndex, item.Node().heightInRecomputeHeap, item.Node().height)
			}
			if _, ok := rh.lookup[item.Node().id]; !ok {
				return fmt.Errorf("recompute heap; sanity check; at height %d item seen that does not exist in recompute heap", heightIndex)
			}
		}
	}
	return nil
}

func (rh *recomputeHeap) String() string {
	output := new(bytes.Buffer)

	fmt.Fprintf(output, "{\n")
	for heightIndex, heightList := range rh.heights {
		if heightList == nil {
			continue
		}
		fmt.Fprintf(output, "\t%d: [", heightIndex)
		lineParts := make([]string, 0, len(heightList))
		for _, li := range heightList {
			lineParts = append(lineParts, fmt.Sprint(li))
		}
		fmt.Fprintf(output, "%s],\n", strings.Join(lineParts, ", "))
	}
	fmt.Fprintf(output, "}\n")
	return output.String()
}
