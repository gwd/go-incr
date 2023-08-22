package incr

import (
	"context"
	"fmt"
)

// Bind lets you swap out an entire subgraph of a computation based
// on a given function and a single input.
//
// A way to think about this, as a sequence:
//
// A given node `a` can be bound to `c` or `d` or more subnodes
// with the value of `a` as the input:
//
//	a -> b.bind() -> c
//
// We might want to, at some point in the future, swap out `c` for `d`
// based on some logic:
//
//	a -> b.bind() -> d
//
// As a result, (a) is a child of (b), and (c) or (d) are children of (b).
// When the bind changes from (c) to (d), (c) is unlinked, and is removed
// as a "child" of (b), preventing it from being considered part of the
// overall computation unless it's referenced by another node in the graph.
func Bind[A, B any](a Incr[A], fn func(A) Incr[B]) BindIncr[B] {
	return BindContext[A, B](a, func(_ context.Context, va A) (Incr[B], error) {
		return fn(va), nil
	})
}

// BindContext is like Bind but allows the bind delegate to take a context and return an error.
//
// If an error returned, the bind is aborted and the error listener(s) will fire for the node.
func BindContext[A, B any](a Incr[A], fn func(context.Context, A) (Incr[B], error)) BindIncr[B] {
	o := &bindIncr[A, B]{
		n:  NewNode(),
		a:  a,
		fn: fn,
		bt: "bind",
	}
	Link(o, a)
	return o
}

// BindIncr is a node that implements Bind, which
// dynamically swaps out entire subgraphs
// based on input incrementals.
type BindIncr[A any] interface {
	Incr[A]
	fmt.Stringer
}

var (
	_ Incr[bool]     = (*bindIncr[string, bool])(nil)
	_ BindIncr[bool] = (*bindIncr[string, bool])(nil)
	_ INode          = (*bindIncr[string, bool])(nil)
	_ fmt.Stringer   = (*bindIncr[string, bool])(nil)
)

type bindIncr[A, B any] struct {
	n     *Node
	bt    string
	a     Incr[A]
	fn    func(context.Context, A) (Incr[B], error)
	bound Incr[B]
}

func (b *bindIncr[A, B]) Node() *Node { return b.n }

func (b *bindIncr[A, B]) Value() (output B) {
	if b.bound != nil {
		output = b.bound.Value()
	}
	return
}

func (b *bindIncr[A, B]) Bind(ctx context.Context) error {
	oldIncr := b.bound
	newIncr, err := b.fn(ctx, b.a.Value())
	if err != nil {
		return err
	}
	var bindChanged bool
	if oldIncr != nil && newIncr != nil {
		if oldIncr.Node().id != newIncr.Node().id {
			bindChanged = true
			b.unlinkOld(oldIncr)
			b.linkNew(newIncr)
		}
	} else if newIncr != nil {
		bindChanged = true
		b.linkNew(newIncr)
	} else if oldIncr != nil {
		bindChanged = true
		b.unlinkOld(oldIncr)
	}
	if bindChanged {
		b.n.boundAt = b.n.graph.stabilizationNum
	}
	return nil
}

func (b *bindIncr[A, B]) unlinkOld(oldIncr INode) {
	for _, p := range b.n.parents {
		p.Node().removeChild(oldIncr.Node().id)
		oldIncr.Node().removeParent(p.Node().id)
	}
	if oldIncr.Node().isOrphaned() {
		b.Node().graph.undiscoverAllNodes(oldIncr)
	}
	b.bound = nil
}

func (b *bindIncr[A, B]) linkNew(newIncr Incr[B]) {
	// for each of the nodes that have the bind node as an input
	// link the new incremental as an input as well (i.e. the bind node
	// itself and the "bound" node are peers in a way).
	// we do this mostly to keep the heights from getting out of control
	// but we could easily have the bind node as part of the chain directly.
	for _, p := range b.n.parents {
		Link(p, newIncr)
		p.Node().recomputeParentHeightsOnBindChange()
	}
	b.Node().graph.discoverAllNodes(newIncr)
	newIncr.Node().changedAt = b.Node().graph.stabilizationNum
	b.Node().graph.recomputeHeap.Add(newIncr)
	b.bound = newIncr
}

func (b *bindIncr[A, B]) String() string {
	return b.n.String(b.bt)
}
