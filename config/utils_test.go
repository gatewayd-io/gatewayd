package config

import "testing"

func TestIf(t *testing.T) {
	if If(true, 1, 2) != 1 {
		t.Error("If(true, 1, 2) != 1")
	}
	if If(false, 1, 2) != 2 {
		t.Error("If(false, 1, 2) != 2")
	}
}

func TestExists(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	if !Exists(m, "a") {
		t.Error("Exists(m, \"a\") != true")
	}
	if Exists(m, "c") {
		t.Error("Exists(m, \"c\") != false")
	}
}
