package main

import "testing"

func TestMurmur3_32(t *testing.T) {
	tests := []struct {
		name string
		data string
		seed uint32
		want uint32
	}{
		{name: "empty string seed 0", data: "", seed: 0, want: 0x00000000},
		{name: "empty string seed 1", data: "", seed: 1, want: 0x514e28b7},
		{name: "single null byte seed 0", data: "\x00", seed: 0, want: 0x514e28b7},
		{name: "four bytes seed 0", data: "\x00\x00\x00\x00", seed: 0, want: 0x2362f9de},
		{name: "Hello", data: "Hello", seed: 0, want: 0x12da77c8},
		{name: "Hello world", data: "Hello, world!", seed: 0, want: 0xc0363e43},
		{name: "determinism", data: "test-input", seed: 42, want: murmur3_32([]byte("test-input"), 42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := murmur3_32([]byte(tt.data), tt.seed)
			if got != tt.want {
				t.Errorf("murmur3_32(%q, %d) = 0x%08x, want 0x%08x", tt.data, tt.seed, got, tt.want)
			}
		})
	}
}

func TestMurmur3_32_DifferentInputsDiffer(t *testing.T) {
	a := murmur3_32([]byte("input-a"), 0)
	b := murmur3_32([]byte("input-b"), 0)
	if a == b {
		t.Errorf("different inputs produced same hash: 0x%08x", a)
	}
}
