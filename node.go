package incr

import (
	"context"
	"fmt"
	"sync"
)

// NewNode returns a new node.
func NewNode() *Node {
	return &Node{
		id:        NewIdentifier(),
		parents:   newNodeList(),
		children:  newNodeList(),
		observers: make(map[Identifier]IObserver),
		createdIn: make(map[Identifier]*bindScope),
	}
}

// Node is the common metadata for any node in the computation graph.
type Node struct {
	// id is a unique identifier for the node
	id Identifier
	// metadata is any additional metadata a user wants to attach to a node.
	metadata any
	// graph is the graph this node is attached to currently.
	graph *Graph
	// label is a descriptive string for the
	// node, and is set with `SetLabel`
	label string
	// parents are the nodes that this node depends on, that is
	// parents are nodes that this node takes as inputs
	parents *nodeList
	// children are the nodes that depend on this node, that is
	// children take this node as an input
	children *nodeList
	// observers are observer nodes that are attached to this
	// node or its children.
	observers map[Identifier]IObserver
	// observersMu interlocsk access to observers
	observersMu sync.Mutex
	// height is the topological sort pseudo-height of the
	// node and is used to order recomputation
	// it is established when the graph is initialized but
	// can also update if bind nodes change their graphs.
	// largely it represents how many levels of inputs feed into
	// this node, e.g. how many other nodes have to update before
	// this node has to update.
	height int
	// changedAt connotes when the node was changed last,
	// specifically if any of the node's parents were set or bound
	changedAt uint64
	// setAt connotes when the node was set last, specifically
	// for var nodes so that we can track their "changed" state separately
	// from their set state
	setAt uint64
	// boundAt connotes when the node was bound last, specifically
	// for bind nodes so that we can track their changed state separately
	// from their bound state
	boundAt uint64
	// recomputedAt connotes when the node was last stabilized
	recomputedAt uint64
	// onUpdateHandlers are functions that are called when the node updates.
	// they are added with `OnUpdate(...)`.
	onUpdateHandlers []func(context.Context)
	// onErrorHandlers are functions that are called when the node updates.
	// they are added with `OnUpdate(...)`.
	onErrorHandlers []func(context.Context, error)
	// onObservedHandlers are functions that are called when the node is observed.
	// they are added with `OnObserved(...)`.
	onObservedHandlers []func(IObserver)
	// onUnobservedHandlers are functions that are called when the node is unobserved.
	// they are added with `OnUnobserved(...)`.
	onUnobservedHandlers []func(IObserver)
	// stabilize is set during initialization and is a shortcut
	// to the interface sniff for the node for the IStabilize interface.
	stabilize func(context.Context) error
	// cutoff is set during initialization and is a shortcut
	// to the interface sniff for the node for the ICutoff interface.
	cutoff func(context.Context) (bool, error)
	// always determines if we always recompute this node.
	always bool
	// numRecomputes is the number of times we recomputed the node
	numRecomputes uint64
	// numChanges is the number of times we changed the node
	numChanges uint64
	// createdIn is the "bind scope" the node was created in
	createdIn map[Identifier]*bindScope
}

func nodeSorter(a, b INode) int {
	if a.Node().height == b.Node().height {
		aID := a.Node().ID().String()
		bID := b.Node().ID().String()
		if aID == bID {
			return 0
		} else if aID > bID {
			return -1
		}
		return 1
	} else if a.Node().height > b.Node().height {
		return -1
	}
	return 1
}

//
// Readonly properties
//

// ID returns a unique identifier for the node.
func (n *Node) ID() Identifier {
	return n.id
}

// String returns a string form of the node metadata.
func (n *Node) String(nodeType string) string {
	if n.label != "" {
		return fmt.Sprintf("%s[%s]:%s@%d", nodeType, n.id.Short(), n.label, n.height)
	}
	return fmt.Sprintf("%s[%s]@%d", nodeType, n.id.Short(), n.height)
}

// Set/Get properties

// OnUpdate registers an update handler.
func (n *Node) OnUpdate(fn func(context.Context)) {
	n.onUpdateHandlers = append(n.onUpdateHandlers, fn)
}

// OnError registers an error handler.
func (n *Node) OnError(fn func(context.Context, error)) {
	n.onErrorHandlers = append(n.onErrorHandlers, fn)
}

// OnObserved registers an observed handler.
func (n *Node) OnObserved(fn func(IObserver)) {
	n.onObservedHandlers = append(n.onObservedHandlers, fn)
}

// OnUnobserved registers an unobserved handler.
func (n *Node) OnUnobserved(fn func(IObserver)) {
	n.onUnobservedHandlers = append(n.onUnobservedHandlers, fn)
}

// Label returns a descriptive label for the node or
// an empty string if one hasn't been provided.
func (n *Node) Label() string {
	return n.label
}

// SetLabel sets the descriptive label on the node.
func (n *Node) SetLabel(label string) {
	n.label = label
}

// Metadata returns user assignable metadata.
func (n *Node) Metadata() any {
	return n.metadata
}

// SetMetadata sets the metadata on the node.
func (n *Node) SetMetadata(md any) {
	n.metadata = md
}

// Parent / Child helpers

// Parents returns the node parent list.
func (n *Node) Parents() []INode {
	return n.parents.Values()
}

// Parents returns the node child list.
func (n *Node) Children() []INode {
	return n.children.Values()
}

// HasChild returns if a child with a given identifier
// is present in the children list.
func (n *Node) HasChild(id Identifier) (ok bool) {
	ok = n.children.HasKey(id)
	return
}

// HasParent returns if a parent with a given identifier
// is present in the parents list.
func (n *Node) HasParent(id Identifier) (ok bool) {
	ok = n.parents.HasKey(id)
	return
}

// IsRoot should return if the parent count, or the
// number of nodes that this node depends on is zero.
func (n *Node) IsRoot() bool {
	return n.parents.IsEmpty()
}

// IsLeaf should return if the child count, or the
// number of nodes depend on this node is zero.
func (n *Node) IsLeaf() bool {
	return n.children.IsEmpty()
}

// Observers returns the node observer list.
func (n *Node) Observers() (output []IObserver) {
	n.observersMu.Lock()
	defer n.observersMu.Unlock()
	output = make([]IObserver, 0, len(n.observers))
	for _, o := range n.observers {
		output = append(output, o)
	}
	return
}

// HasObserver returns if an observer with a given identifier
// is present in the observers list.
func (n *Node) HasObserver(id Identifier) (ok bool) {
	n.observersMu.Lock()
	defer n.observersMu.Unlock()
	_, ok = n.observers[id]
	return
}

//
// Internal Helpers
//

// addChildren adds node references as children to this node.
func (n *Node) addChildren(c ...INode) {
	n.children.Push(c...)
}

// addParents adds node references as parents to this node.
func (n *Node) addParents(c ...INode) {
	n.parents.Push(c...)
}

// addObservers adds observers to the node, calling
// onObserved handlers with a separate invocation
// for each observer that is added.
func (n *Node) addObservers(observers ...IObserver) {
	n.observersMu.Lock()
	defer n.observersMu.Unlock()
	for _, o := range observers {
		n.observers[o.Node().id] = o
		for _, handler := range n.onObservedHandlers {
			handler(o)
		}
	}
}

// RemoveChild removes a specific child from the node, specifically
// a node that might have been an input to this node.
func (n *Node) removeChild(id Identifier) {
	n.children.RemoveKey(id)
}

// RemoveParent removes a parent from the node, specifically
// a node for which this node is an input.
func (n *Node) removeParent(id Identifier) {
	n.parents.RemoveKey(id)
}

// maybeCutoff calls the cutoff delegate if it's set, otherwise
// just returns false (effectively _not_ cutting off the computation).
func (n *Node) maybeCutoff(ctx context.Context) (bool, error) {
	if n.cutoff != nil {
		return n.cutoff(ctx)
	}
	return false, nil
}

// detectCutoff detects if a INode (which should be the same
// as as managed by this node reference), implements ICutoff
// and grabs a reference to the Cutoff delegate function.
func (n *Node) detectCutoff(gn INode) {
	if typed, ok := gn.(ICutoff); ok {
		n.cutoff = typed.Cutoff
	}
}

// detectAlways detects if a INode (which should be the same
// as as managed by this node reference), implements IAlways.
func (n *Node) detectAlways(gn INode) {
	_, n.always = gn.(IAlways)
}

// detectStabilize detects if a INode (which should be the same
// as as managed by this node reference), implements IStabilize
// and grabs a reference to the Stabilize delegate function.
func (n *Node) detectStabilize(gn INode) {
	if typed, ok := gn.(IStabilize); ok {
		n.stabilize = typed.Stabilize
	}
}

// ShouldRecompute returns whether or not a given node needs to be recomputed.
func (n *Node) ShouldRecompute() bool {
	// we should always recompute on the first stabilization
	if n.recomputedAt == 0 {
		return true
	}
	if n.always {
		return true
	}

	// if a node can't stabilize, return false
	if n.stabilize == nil {
		return false
	}

	// if the node was marked stale explicitly
	// either because it is a var or was
	// called as a parameter to `graph.SetStale`
	if n.setAt > n.recomputedAt {
		return true
	}
	// if the node had a bind change recently
	if n.boundAt > n.recomputedAt {
		return true
	}
	if n.changedAt > n.recomputedAt {
		return true
	}

	// if any of the direct _inputs_ to this node have changed
	// or updated their bind. we don't go full recursive
	// here to prevent a bunch of extra work.
	n.parents.Lock()
	defer n.parents.Unlock()
	for _, p := range n.parents.list.items {
		if p.value.Node().changedAt > n.recomputedAt || p.value.Node().boundAt > n.recomputedAt {
			return true
		}
	}
	return false
}

// computePseudoHeight calculates the nodes height in respect to its parents.
//
// it will use the maximum height _the node has ever seen_, i.e.
// if the height is 1, then 3, then 1 again, this will return 3.
func (n *Node) computePseudoHeight() int {
	var maxParentHeight int
	var parentHeight int

	n.parents.Each(func(p INode) {
		parentHeight = p.Node().computePseudoHeight()
		if parentHeight > maxParentHeight {
			maxParentHeight = parentHeight
		}
	})

	// we do this to prevent the height
	// changing a bunch with bind nodes.
	// basically just stick with the overall maximum
	// height the node has seen ever.
	if n.height > maxParentHeight {
		return n.height
	}
	return maxParentHeight + 1
}

func (n *Node) maybeStabilize(ctx context.Context) (err error) {
	if n.stabilize != nil {
		if err = n.stabilize(ctx); err != nil {
			return
		}
	}
	return
}
