package incr

import (
	"context"
	"testing"

	"github.com/wcharczuk/go-incr/testutil"
)

func Test_Observe_Unobserve(t *testing.T) {
	ctx := testContext()
	g := New()

	v0 := Var(ctx, "hello 0")
	m0 := Map(ctx, v0, ident)

	v1 := Var(ctx, "hello 1")
	m1 := Map(ctx, v1, ident)

	o0 := Observe(ctx, g, m0)
	o1 := Observe(ctx, g, m1)

	testutil.ItsEqual(t, 6, g.numNodes)

	testutil.ItsEqual(t, true, g.IsObserving(m0))
	testutil.ItsEqual(t, true, g.IsObserving(m1))

	testutil.ItsEqual(t, "", o0.Value())
	testutil.ItsEqual(t, "", o1.Value())

	err := g.Stabilize(context.TODO())
	testutil.ItsNil(t, err)

	testutil.ItsEqual(t, "hello 0", o0.Value())
	testutil.ItsEqual(t, "hello 1", o1.Value())

	o1.Unobserve(ctx)

	testutil.ItsEqual(t, g.observed.Len(), g.numNodes-1, "we don't observe the observer but we do track it!")
	testutil.ItsNil(t, o1.Node().graph)

	// should take effect immediately because there is only (1) observer.
	testutil.ItsEqual(t, true, g.IsObserving(m0))
	testutil.ItsEqual(t, false, g.IsObserving(m1))

	v0.Set("not hello 0")
	v1.Set("not hello 1")
	err = g.Stabilize(context.TODO())
	testutil.ItsNil(t, err)

	testutil.ItsEqual(t, "not hello 0", o0.Value())
	testutil.ItsEqual(t, "", o1.Value())
}

func Test_Observe_Unobserve_multiple(t *testing.T) {
	ctx := testContext()
	g := New()

	v0 := Var(ctx, "hello 0")
	m0 := Map(ctx, v0, ident)

	v1 := Var(ctx, "hello 1")
	m1 := Map(ctx, v1, ident)

	o0 := Observe(ctx, g, m0)
	o1 := Observe(ctx, g, m1)
	o11 := Observe(ctx, g, m1)

	testutil.ItsEqual(t, true, g.IsObserving(v0))
	testutil.ItsEqual(t, true, g.IsObserving(m0))
	testutil.ItsEqual(t, true, g.IsObserving(v1))
	testutil.ItsEqual(t, true, g.IsObserving(m1))

	testutil.ItsEqual(t, 1, len(v0.Node().Observers()))
	testutil.ItsEqual(t, 1, len(m0.Node().Observers()))
	testutil.ItsEqual(t, 2, len(v1.Node().Observers()))
	testutil.ItsEqual(t, 2, len(m1.Node().Observers()))

	testutil.ItsEqual(t, "", o0.Value())
	testutil.ItsEqual(t, "", o1.Value())
	testutil.ItsEqual(t, "", o11.Value())

	err := g.Stabilize(context.TODO())
	testutil.ItsNil(t, err)

	testutil.ItsEqual(t, "hello 0", o0.Value())
	testutil.ItsEqual(t, "hello 1", o1.Value())
	testutil.ItsEqual(t, "hello 1", o11.Value())

	o1.Unobserve(ctx)

	testutil.ItsEqual(t, g.observed.Len(), g.numNodes-2, "we should have (1) less observer after unobserve!")
	testutil.ItsNil(t, o1.Node().graph)

	testutil.ItsEqual(t, 0, o1.Node().parents.Len())
	testutil.ItsEqual(t, 0, o1.Node().children.Len())
	testutil.ItsNone(t, m1.Node().Children(), func(n INode) bool {
		return n.Node().ID() == o1.Node().ID()
	})

	testutil.ItsEqual(t, true, g.IsObserving(m0))
	testutil.ItsEqual(t, true, g.IsObserving(m1))

	testutil.ItsEqual(t, 1, len(v0.Node().Observers()))
	testutil.ItsEqual(t, 1, len(m0.Node().Observers()))
	testutil.ItsEqual(t, 1, len(v1.Node().Observers()))
	testutil.ItsEqual(t, 1, len(m1.Node().Observers()))

	v0.Set("not hello 0")
	v1.Set("not hello 1")
	err = g.Stabilize(context.TODO())
	testutil.ItsNil(t, err)

	testutil.ItsEqual(t, "not hello 0", o0.Value())
	testutil.ItsEqual(t, "", o1.Value())
	testutil.ItsEqual(t, "not hello 1", o11.Value())
}

func Test_Observer_Unobserve_reobserve(t *testing.T) {
	ctx := testContext()
	g := New()
	v0 := Var(ctx, "hello")
	m0 := Map(ctx, v0, ident)
	o0 := Observe(ctx, g, m0)

	_ = g.Stabilize(context.TODO())
	testutil.ItsEqual(t, "hello", o0.Value())

	o0.Unobserve(ctx)

	_ = g.Stabilize(context.TODO())
	testutil.ItsEqual(t, false, g.IsObserving(m0))
	// strictly, the value shouldn't change ...
	testutil.ItsEqual(t, "hello", m0.Value())

	o1 := Observe(ctx, g, m0)
	_ = g.Stabilize(context.TODO())
	testutil.ItsEqual(t, "hello", o1.Value())
}
