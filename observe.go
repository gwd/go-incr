package incr

import (
	"context"
	"fmt"
)

// Observe observes a node, specifically including it for computation
// as well as all of its parents.
func Observe[A any](g *Graph, input Incr[A]) ObserveIncr[A] {
	o := &observeIncr[A]{
		n:     NewNode(),
		input: input,
	}
	Link(o, input)
	g.DiscoverObserver(o)

	// NOTE(wc): we do this here because some """expert""" use cases
	// require us to "discover" observers but _not_ add them to the recompute heap.
	// So we just add it here explicitly and don't add it implicitly
	// in the DiscoverObserver function.
	g.recomputeHeap.Add(o)
	g.DiscoverNodes(o, input)
	return o
}

// ObserveIncr is an incremental that observes a graph
// of incrementals starting a given input.
type ObserveIncr[A any] interface {
	Incr[A]
	Unobserve(context.Context)
}

// IObserver is an INode that can be unobserved.
type IObserver interface {
	INode
	Unobserve(context.Context)
}

var (
	_ Incr[any]    = (*observeIncr[any])(nil)
	_ INode        = (*observeIncr[any])(nil)
	_ fmt.Stringer = (*observeIncr[any])(nil)
)

type observeIncr[A any] struct {
	n     *Node
	input Incr[A]
}

func (o *observeIncr[A]) Node() *Node { return o.n }

// Unobserve effectively removes a given node
// from the observed ref count for a graph.
func (o *observeIncr[A]) Unobserve(ctx context.Context) {
	g := o.n.graph
	g.UndiscoverNodes(ctx, o, o.input)
	g.UndiscoverObserver(o)
	o.input = nil
}

func (o *observeIncr[A]) Value() (output A) {
	if o.input != nil {
		output = o.input.Value()
	}
	return
}

func (o *observeIncr[A]) String() string {
	return o.n.String("observer")
}
