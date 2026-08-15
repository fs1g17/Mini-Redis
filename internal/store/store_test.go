package store

import (
	"testing"
	"time"
)

func Test_Set(t *testing.T) {
	store := NewRedisStore()

	store.Set("foo", "bar")
	if len(store.data) != 1 || store.data["foo"] != "bar" {
		t.Fatal("expected 'set' to correctly set data")
	}

	store.Set("foo", "car")
	if len(store.data) != 1 || store.data["foo"] != "car" {
		t.Fatal("expected 'set' to correctly set data")
	}
}

func Test_Get(t *testing.T) {
	store := NewRedisStore()
	store.data["foo"] = "bar"

	value, exists := store.Get("foo")
	if value != "bar" || !exists {
		t.Fatal("expected get to work on defined values")
	}

	value, exists = store.Get("boo")
	if value != "" || exists {
		t.Fatal("expected get to work on undefined values")
	}
}

func Test_Del(t *testing.T) {
	store := NewRedisStore()
	store.data["foo"] = "bar"
	store.data["foo2"] = "bar2"
	store.expire["foo"] = time.Now().Unix() + int64(1000)

	count := store.Del("foo3")
	if count != 0 {
		t.Fatal("expected to not delete anything")
	}

	count = store.Del("foo", "foo2")
	if count != 2 {
		t.Fatal("expected to delete 2 keys")
	}

	if len(store.data) != 0 || len(store.expire) != 0 {
		t.Fatal("expected data and expire to be empty")
	}
}

func Test_Exists(t *testing.T) {
	now := time.Now().Unix()
	store := NewRedisStore()
	store.data["foo"] = "bar"
	store.data["foo2"] = "bar2"
	store.expire["foo2"] = now

	count := store.Exists("foo", "foo2")
	if count != 1 {
		t.Fatal("expected exists count to be 1")
	}
}

func Test_Expire(t *testing.T) {
	store := NewRedisStore()
	store.data["foo"] = "bar"

	ok := store.Expire("foo2", 20)
	if ok {
		t.Fatal("expected to fail expiring non-existent key")
	}

	ok = store.Expire("foo", 20)
	if !ok {
		t.Fatal("expected to succeed expiring existing key")
	}

	_, ok = store.expire["foo"]
	if !ok {
		t.Fatal("expected expiry present on the correct key")
	}
}
