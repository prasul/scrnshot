package main

import "testing"

func TestClampIndex(t *testing.T) {
	cases := []struct{ in, n, want int }{
		{0, 5, 0},
		{4, 5, 4},
		{5, 5, 0},  // past the end wraps to top
		{-1, 5, 4}, // before the start wraps to bottom
		{0, 0, 0},  // empty list
	}
	for _, c := range cases {
		if got := clampIndex(c.in, c.n); got != c.want {
			t.Errorf("clampIndex(%d,%d)=%d want %d", c.in, c.n, got, c.want)
		}
	}
}

func TestIndexOfKey(t *testing.T) {
	items := []menuItem{{"1", "a"}, {"2", "b"}, {"q", "quit"}}
	if i := indexOfKey(items, "2"); i != 1 {
		t.Errorf("indexOfKey 2 = %d want 1", i)
	}
	if i := indexOfKey(items, "q"); i != 2 {
		t.Errorf("indexOfKey q = %d want 2", i)
	}
	if i := indexOfKey(items, "9"); i != -1 {
		t.Errorf("indexOfKey 9 = %d want -1", i)
	}
}

func TestRunTUINonInteractive(t *testing.T) {
	// In test (no TTY on stdin), runTUI must refuse rather than hang on stty.
	if err := runTUI(defaultConfig()); err == nil {
		t.Error("expected runTUI to error without a TTY")
	}
}
