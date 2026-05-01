package adapter

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dlf-dds/goat-translator/internal/canonical"
)

type fakeAdapter struct {
	name string
}

func (f *fakeAdapter) Name() string                                    { return f.name }
func (f *fakeAdapter) Description() string                             { return "fake " + f.name }
func (f *fakeAdapter) Decode(_ []byte) (canonical.Entity, error)       { return canonical.Entity{}, nil }
func (f *fakeAdapter) Encode(_ canonical.Entity) ([]byte, error)       { return nil, nil }
func (f *fakeAdapter) Detect(_ []byte) bool                            { return false }

func TestRegisterAndGet(t *testing.T) {
	ResetForTest()
	a := &fakeAdapter{name: "fake"}
	Register(a)

	got, err := Get("fake")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != a {
		t.Fatalf("Get returned wrong adapter: got %v want %v", got, a)
	}
}

func TestGetUnknown(t *testing.T) {
	ResetForTest()
	_, err := Get("nope")
	if !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("expected ErrUnknownFormat, got %v", err)
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	ResetForTest()
	Register(&fakeAdapter{name: "dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	Register(&fakeAdapter{name: "dup"})
}

func TestRegisterNilPanics(t *testing.T) {
	ResetForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil adapter")
		}
	}()
	Register(nil)
}

func TestRegisterEmptyNamePanics(t *testing.T) {
	ResetForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty name")
		}
	}()
	Register(&fakeAdapter{name: ""})
}

func TestListSorted(t *testing.T) {
	ResetForTest()
	Register(&fakeAdapter{name: "charlie"})
	Register(&fakeAdapter{name: "alpha"})
	Register(&fakeAdapter{name: "bravo"})

	got := List()
	want := []string{"alpha", "bravo", "charlie"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List returned %v, want %v", got, want)
	}
}

func TestAllReturnsEveryAdapter(t *testing.T) {
	ResetForTest()
	a := &fakeAdapter{name: "alpha"}
	b := &fakeAdapter{name: "bravo"}
	Register(a)
	Register(b)

	all := All()
	if len(all) != 2 {
		t.Fatalf("All returned %d adapters, want 2", len(all))
	}
	if all[0] != a || all[1] != b {
		t.Fatalf("All returned wrong order: %v", all)
	}
}
