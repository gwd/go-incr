package incr

import (
	"context"
	"fmt"
)

// Bind lets you swap out an entire subgraph of a computation based
// on a given function and a single input.
//
// A way to think about this, as a sequence:
//    a -> b.bind() -> (c | d | ...)
//    a -> b.bind() -> c
//    a -> b.bind() -> ~c~ d
//
// As a result, (a) is a child of (b), and (c) or (d) are children of (b).
// When the bind changes from (c) to (d), (c) is unlinked, and is removed
// as a "child" of (b),
func Bind[A, B any](a Incr[A], fn func(A) Incr[B]) BindIncr[B] {
	o := &bindIncr[A, B]{
		n: NewNode(),
		a: a,
		fn: func(_ context.Context, va A) (Incr[B], error) {
			return fn(va), nil
		},
	}
	Link(o, a)
	return o
}

// BindContext is like Bind but takes a context and returns an error for
// the bind delegate itself.
func BindContext[A, B any](a Incr[A], fn func(context.Context, A) (Incr[B], error)) BindIncr[B] {
	o := &bindIncr[A, B]{
		n:  NewNode(),
		a:  a,
		fn: fn,
	}
	Link(o, a)
	return o
}

// BindIncr is a node that implements Bind, which
// dynamically swaps out entire subgraphs
// based on input incrementals.
type BindIncr[A any] interface {
	Incr[A]
	Bind(context.Context) (old, new Incr[A], err error)
	SetBind(Incr[A])
}

var (
	_ Incr[bool]     = (*bindIncr[string, bool])(nil)
	_ BindIncr[bool] = (*bindIncr[string, bool])(nil)
	_ INode          = (*bindIncr[string, bool])(nil)
	_ IStabilize     = (*bindIncr[string, bool])(nil)
	_ fmt.Stringer   = (*bindIncr[string, bool])(nil)
)

type bindIncr[A, B any] struct {
	n     *Node
	a     Incr[A]
	fn    func(context.Context, A) (Incr[B], error)
	bound Incr[B]
	value B
}

func (b *bindIncr[A, B]) Node() *Node { return b.n }

func (b *bindIncr[A, B]) Value() B { return b.value }

func (b *bindIncr[A, B]) SetBind(v Incr[B]) {
	b.bound = v
}

func (b *bindIncr[A, B]) Bind(ctx context.Context) (oldValue, newValue Incr[B], err error) {
	oldValue = b.bound
	newValue, err = b.fn(ctx, b.a.Value())
	return
}

func (b *bindIncr[A, B]) Stabilize(ctx context.Context) error {
	if err := bindUpdate[B](ctx, b); err != nil {
		return err
	}
	b.value = b.bound.Value()
	return nil
}

func (b *bindIncr[A, B]) String() string {
	return Label(b.n, "bind")
}

// bindUpdate is a helper for dealing with bind node changes
// specifically handling unlinking and linking bound nodes
// when the bind changes.
func bindUpdate[A any](ctx context.Context, b BindIncr[A]) error {
	g := b.Node().g

	oldIncr, newIncr, err := b.Bind(ctx)
	if err != nil {
		return err
	}

	if oldIncr == nil {
		// link the new value as the parent
		// of the bind node, specifically
		// that b is an input to newValue
		Link(newIncr, b)
		discoverAllNodes(ctx, g, newIncr)
		b.SetBind(newIncr)
		newIncr.Node().changedAt = g.stabilizationNum
		return newIncr.Node().maybeStabilize(ctx)
	}

	if oldIncr.Node().id != newIncr.Node().id {
		// unlink the old node from the bind node
		b.Node().parents = nil
		oldIncr.Node().children = nil
		undiscoverAllNodes(ctx, g, oldIncr)

		// link the new value as the parent
		// of the bind node, specifically
		// that b is an input to newValue
		Link(newIncr, b)
		discoverAllNodes(ctx, g, newIncr)
		b.SetBind(newIncr)
		newIncr.Node().changedAt = g.stabilizationNum
		return newIncr.Node().maybeStabilize(ctx)
	}
	return nil
}
