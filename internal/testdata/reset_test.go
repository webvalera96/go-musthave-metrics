package testdata

import "testing"

func TestResetableStruct_Reset_nilReceiver(t *testing.T) {
	var r *ResetableStruct
	r.Reset()
}

func TestResetableStruct_Reset_resetsFieldsAndChild(t *testing.T) {
	t.Helper()

	str := "value"
	childStr := "child"

	r := &ResetableStruct{
		I:    42,
		Str:  "hello",
		StrP: &str,
		S:    []int{1, 2, 3},
		M: map[string]string{
			"a": "b",
			"c": "d",
		},
		Child: &ResetableStruct{
			I:    7,
			Str:  "child",
			StrP: &childStr,
			S:    []int{9, 8},
			M: map[string]string{
				"x": "y",
			},
		},
	}

	r.Reset()

	if r.I != 0 {
		t.Fatalf("I: expected 0, got %d", r.I)
	}
	if r.Str != "" {
		t.Fatalf("Str: expected empty, got %q", r.Str)
	}
	if str != "" {
		t.Fatalf("StrP: expected pointed value empty, got %q", str)
	}
	if r.S == nil || len(r.S) != 0 {
		t.Fatalf("S: expected len 0 (non-nil), got len=%d nil=%v", len(r.S), r.S == nil)
	}
	if len(r.M) != 0 {
		t.Fatalf("M: expected cleared map, got len=%d", len(r.M))
	}
	if r.Child == nil {
		t.Fatalf("Child: expected non-nil")
	}
	if r.Child.I != 0 || r.Child.Str != "" || childStr != "" || len(r.Child.S) != 0 || len(r.Child.M) != 0 {
		t.Fatalf("Child: expected fully reset, got %+v (childStr=%q)", *r.Child, childStr)
	}
}
