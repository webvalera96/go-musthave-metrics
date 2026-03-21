package configfile

import (
	"reflect"
	"testing"
)

func TestFilterArgs(t *testing.T) {
	in := []string{"./agent", "-c", "/tmp/cfg.json", "-a", "localhost:9090", "-p", "3"}
	got := FilterArgs(in)
	want := []string{"./agent", "-a", "localhost:9090", "-p", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterArgs: got %v, want %v", got, want)
	}
}

func TestFilterArgsEqualsForm(t *testing.T) {
	in := []string{"./server", "-config=/x.json", "-a", ":1"}
	got := FilterArgs(in)
	want := []string{"./server", "-a", ":1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterArgs: got %v, want %v", got, want)
	}
}
