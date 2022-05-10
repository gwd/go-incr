package incr

import (
	"context"
	"fmt"
)

// DiffMapByKeys returns two incrementals, one for keys added, and one
// for keys removed, and each stabilization pass returns just the subset
// of the map that changed since the last pass according to the keys.
func DiffMapByKeys[K comparable, V any](i Incr[map[K]V]) (add Incr[map[K]V], rem Incr[map[K]V]) {
	add = &diffMapByKeysAddedIncr[K, V]{
		n: NewNode(),
		i: i,
	}
	Link(add, i)
	rem = &diffMapByKeysRemovedIncr[K, V]{
		n: NewNode(),
		i: i,
	}
	Link(add, i)
	return
}

// DiffMapByKeysAdded returns an incremental that takes an input map typed
// incremental, and each stabilization pass returns just the subset
// of the map that was added since the last pass according to the keys.
func DiffMapByKeysAdded[K comparable, V any](i Incr[map[K]V]) Incr[map[K]V] {
	o := &diffMapByKeysAddedIncr[K, V]{
		n: NewNode(),
		i: i,
	}
	Link(o, i)
	return o
}

// DiffMapByKeysRemoved returns an incremental that takes an input map typed
// incremental, and each stabilization pass returns just the subset
// of the map that was removed since the last pass according to the keys.
func DiffMapByKeysRemoved[K comparable, V any](i Incr[map[K]V]) Incr[map[K]V] {
	o := &diffMapByKeysRemovedIncr[K, V]{
		n: NewNode(),
		i: i,
	}
	Link(o, i)
	return o
}

var (
	_ Incr[map[string]int] = (*diffMapByKeysAddedIncr[string, int])(nil)
	_ INode                = (*diffMapByKeysAddedIncr[string, int])(nil)
	_ IStabilize           = (*diffMapByKeysAddedIncr[string, int])(nil)
	_ fmt.Stringer         = (*diffMapByKeysAddedIncr[string, int])(nil)
)

type diffMapByKeysAddedIncr[K comparable, V any] struct {
	n   *Node
	i   Incr[map[K]V]
	val map[K]V
}

func (mfn *diffMapByKeysAddedIncr[K, V]) String() string {
	return FormatNode(mfn.n, "map_diff_key_adds")
}

func (mfn *diffMapByKeysAddedIncr[K, V]) Node() *Node { return mfn.n }

func (mfn *diffMapByKeysAddedIncr[K, V]) Value() map[K]V { return mfn.val }

func (mfn *diffMapByKeysAddedIncr[K, V]) Stabilize(_ context.Context) error {
	mfn.val = diffMapByKeysAdded(mfn.val, mfn.i.Value())
	return nil
}

var (
	_ Incr[map[string]int] = (*diffMapByKeysRemovedIncr[string, int])(nil)
	_ INode                = (*diffMapByKeysRemovedIncr[string, int])(nil)
	_ IStabilize           = (*diffMapByKeysRemovedIncr[string, int])(nil)
	_ fmt.Stringer         = (*diffMapByKeysRemovedIncr[string, int])(nil)
)

type diffMapByKeysRemovedIncr[K comparable, V any] struct {
	n   *Node
	i   Incr[map[K]V]
	val map[K]V
}

func (mfn *diffMapByKeysRemovedIncr[K, V]) String() string {
	return FormatNode(mfn.n, "diff_map_by_keys_removed")
}

func (mfn *diffMapByKeysRemovedIncr[K, V]) Node() *Node { return mfn.n }

func (mfn *diffMapByKeysRemovedIncr[K, V]) Value() map[K]V { return mfn.val }

func (mfn *diffMapByKeysRemovedIncr[K, V]) Stabilize(_ context.Context) error {
	mfn.val = diffMapByKeysRemoved(mfn.val, mfn.i.Value())
	return nil
}

func diffMapByKeysAdded[K comparable, V any](m0, m1 map[K]V) (add map[K]V) {
	add = make(map[K]V)
	var ok bool
	if m0 != nil {
		for k, v := range m1 {
			if _, ok = m0[k]; !ok {
				add[k] = v
			}
		}
		return
	}
	for k, v := range m1 {
		add[k] = v
	}
	return
}

func diffMapByKeysRemoved[K comparable, V any](m0, m1 map[K]V) (rem map[K]V) {
	rem = make(map[K]V)
	var ok bool
	if m1 != nil {
		for k, v := range m0 {
			if _, ok = m1[k]; !ok {
				rem[k] = v
			}
		}
		return
	}
	if m0 != nil {
		for k, v := range m0 {
			rem[k] = v
		}
		return
	}
	return
}