package syncmap

import (
	"slices"
	"testing"
)

func TestMap_Items_AllItems(t *testing.T) {
	m := Map[string, string]{}
	m.Store("a", "b")
	m.Store("foo", "bar")

	keysSeen := []string{}
	for key, value := range m.Items() {
		keysSeen = append(keysSeen, key)
		if !slices.Contains([]string{"a", "foo"}, key) {
			t.Errorf("unknown key: %q", key)
			continue
		}
		switch key {
		case "a":
			if value != "b" {
				t.Errorf("bad value for \"a\": %q != %q", "b", value)
			}
		case "foo":
			if value != "bar" {
				t.Errorf("bad value for \"foo\": %q != %q", "bar", value)
			}
		default:
			t.Errorf("unknown key: %q", key)
		}
	}
	if len(keysSeen) != 2 {
		t.Errorf("expected 2 keys to be iterated on; got %d", len(keysSeen))
	}
	if !slices.Contains(keysSeen, "a") {
		t.Errorf("key not iterated on: %q", "a")
	}
	if !slices.Contains(keysSeen, "foo") {
		t.Errorf("key not iterated on: %q", "foo")
	}
}

func TestMap_Items_ExitEarly(t *testing.T) {
	m := Map[string, string]{}
	m.Store("a", "b")
	m.Store("foo", "bar")
	m.Store("exitearly", "")

	keysSeen := []string{}
	for key := range m.Items() {
		keysSeen = append(keysSeen, key)
		break
	}

	if len(keysSeen) != 1 {
		t.Errorf("expected 1 key to be iterated on; got %d", len(keysSeen))
	}
}
