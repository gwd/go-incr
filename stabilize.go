package incr

import (
	"context"
)

// Stabilize kicks off the full stabilization pass given initial nodes
// representing graphs.
//
// The nodes do not need to be any specific type of node in the graph
// as the full graph will be initialized on the first call to stabilize for that graph.
func (graph *Graph) Stabilize(ctx context.Context) (err error) {
	if err = graph.ensureNotStabilizing(ctx); err != nil {
		return
	}
	ctx = graph.stabilizeStart(ctx)
	defer func() {
		graph.stabilizeEnd(ctx, err)
	}()

	var immediateRecompute []INode
	var next []INode
	for len(graph.recomputeHeap.lookup) > 0 {
		next = graph.recomputeHeap.removeMinHeight()
		for _, n := range next {
			if err = graph.recompute(ctx, n); err != nil {
				break
			}
			if n.Node().always {
				immediateRecompute = append(immediateRecompute, n)
			}
		}
		if err != nil {
			break
		}
		// graph.fixAdjustHeightsQueue()
	}
	graph.recomputeHeap.add(immediateRecompute...)
	return
}
