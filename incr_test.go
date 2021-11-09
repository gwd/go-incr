package incremental

import "testing"

func Test_Return(t *testing.T) {
	expected := "foo"
	n := Return(expected)

	if value := n.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
}

func Test_Map(t *testing.T) {
	m := Map(Return("foo"), func(v string) string {
		return "not "+v
	})

	expected := "not foo"
	if value := m.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
}

func Test_Map2(t *testing.T) {
	m2 := Map2(Return("foo"), Return("bar"), func(v0, v1 string) string {
		return v0+" "+v1
	})

	expected := "foo bar"
	if value := m2.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
}

func Test_MapIf(t *testing.T) {
	expected := "foo"
	mi := MapIf(Return("foo"), Return("bar"), Return(true))

	if value := mi.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}

	expected = "bar"
	mi = MapIf(Return("foo"), Return("bar"), Return(false))

	if value := mi.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
}

func Test_Bind(t *testing.T) {
	b := Bind(Return("foo"), func(v string) Incr[string] {
		return IncrFunc[string](func() string {
			return v + " bar"
		})
	})

	expected := "foo bar"
	if value := b.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
}

func Test_BindIf(t *testing.T) {
	expected := "foo"
	mi := BindIf(Return("foo"), Return("bar"), Return(true))

	if value := mi.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}

	expected = "bar"
	mi = BindIf(Return("foo"), Return("bar"), Return(false))

	if value := mi.Value(); value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
}

func Test_OnUpdate(t *testing.T) {
	var updatedValue string
	o := OnUpdate(Map(Return("foo"), func(v string) string {
		return "not "+v
	}), func(v string) {
		updatedValue = v
	})

	expected := "not foo"
	value := o.Value()
	if value != expected {
		t.Errorf("expected %q, actual: %q", expected, value)
		t.FailNow()
	}
	if updatedValue != expected {
		t.Errorf("expected %q, actual: %q", expected, updatedValue)
		t.FailNow()
	}
}
