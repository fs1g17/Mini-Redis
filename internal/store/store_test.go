package store

import "testing"

func Test_Store(t *testing.T) {
	store := NewRedisStore()

	if len(store.data) != 0 {
		t.Fatal("expected store to initialise empty")
	}

	store.Set("foo", "bar")
	if len(store.data) != 1 || store.data["foo"] != "bar" {
		t.Fatal("expected 'set' to correctly set data")
	}

	value, exists := store.Get("foo")
	if value != "bar" || !exists {
		t.Fatal("expected 'get' to correctly get set data")
	}

	value, exists = store.Get("car")
	if value != "" || exists {
		t.Fatal("expected 'get' to correctly get unset data")
	}

	count := store.Exists("foo", "car")
	if count != 1 {
		t.Fatal("expected 'exists' to correctly get count of keys")
	}

	count = store.Del("foo", "car")
	if count != 1 {
		t.Fatal("expected 'del' to correctly delete and return count of deleted keys")
	}

	if len(store.data) != 0 {
		t.Fatal("expected store to initialise empty")
	}
}
