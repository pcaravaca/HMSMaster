package main

import "testing"

func TestTaskTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"pending", "in_progress", true},
		{"pending", "done", true},
		{"in_progress", "done", true},
		{"done", "in_progress", false},
		{"done", "pending", false},
		{"in_progress", "pending", false},
		{"pending", "unknown", false},
	}
	for _, tc := range cases {
		if got := validTaskTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("%s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
