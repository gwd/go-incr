package incr

// ExpertVar returns an "expert" version of a var node.
//
// Note there are no compatibility guarantees on this interface
// and you should use this interface at your own risk.
func ExpertVar[A any](v VarIncr[A]) IExpertVar[A] {
	return &expertVar[A]{v: v.(*varIncr[A])}
}

// IExpertVar are methods implemented by ExpertVar.
//
// Note there are no compatibility guarantees on this interface
// and you should use this interface at your own risk.
type IExpertVar[A any] interface {
	// SetInternalValue allows you to set the underlying value of a var
	// without marking it as stale.
	//
	// This can be useful when deserializing graphs from some other state.
	SetInternalValue(A)
}

type expertVar[A any] struct {
	v *varIncr[A]
}

func (ev *expertVar[A]) SetInternalValue(v A) {
	ev.v.value = v
}
