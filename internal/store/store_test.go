package store

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

var startTime time.Time = time.Date(2012, 07, 15, 0, 0, 0, 0, time.UTC)
var currTime time.Time = startTime

func Now() time.Time {
	return currTime
}

func Test_Set(t *testing.T) {
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
	store.data["key"] = "value"
	store.expire["key"] = Now().Unix() + 200

	store.Set("foo", "bar")
	if len(store.data) != 2 || store.data["foo"] != "bar" {
		t.Fatal("expected 'set' to correctly set data")
	}

	store.Set("foo", "car")
	if len(store.data) != 2 || store.data["foo"] != "car" {
		t.Fatal("expected 'set' to correctly set data")
	}

	store.Set("key", "newValue")
	if len(store.data) != 2 || store.data["key"] != "newValue" || len(store.expire) != 0 {
		t.Fatal("expected 'set' to correctly clear expiration")
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
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
	store.data["foo"] = "bar"
	store.data["foo2"] = "bar2"
	store.expire["foo"] = Now().Unix() + int64(1000)

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
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
	store.data["foo"] = "bar"
	store.data["foo2"] = "bar2"
	store.expire["foo2"] = startTime.Unix()

	count := store.Exists("foo", "foo2")
	if count != 1 {
		t.Fatal("expected exists count to be 1")
	}
}

func Test_Expire(t *testing.T) {
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
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

func Test_TTL(t *testing.T) {
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
	store.data["foo"] = "bar"
	store.data["foo2"] = "bar2"
	store.expire["foo"] = Now().Unix() + 20

	ttl := store.TTL("foo")
	if ttl != 20 {
		t.Fatal("expected ttl to be 20")
	}

	currTime = startTime.Add(5 * time.Second)
	defer func() {
		currTime = startTime
	}()

	ttl = store.TTL("foo")
	if ttl != 15 {
		t.Fatal("expected ttl to be 15")
	}

	ttl = store.TTL("foo2")
	if ttl != -1 {
		t.Fatal("expected ttl to be -1 for persistent keys")
	}

	ttl = store.TTL("foo3")
	if ttl != -2 {
		t.Fatal("expected ttl to be -2 for non-existent key")
	}
}

func Test_Incr(t *testing.T) {
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
	store.data["foo"] = "12"
	store.data["foo2"] = "bar"

	value, err := store.Incr("new")
	if err != nil {
		t.Fatalf("expected err to be nil, got %v\n", err)
	}

	if value != 0 {
		t.Fatalf("expected 0, got: %d\n", value)
	}

	// --- here ---

	value, err = store.Incr("foo")
	if err != nil {
		t.Fatalf("expected err to be nil, got %v\n", err)
	}

	if value != 13 {
		t.Fatalf("expected 13, got: %d\n", value)
	}
	// --- here ---

	value, err = store.Incr("foo2")
	if err == nil {
		t.Fatalf("epxected err to be not nil, got %v\n", err)
	}

}

func Test_Poop(t *testing.T) {
	store := &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
		now:    Now,
	}
	store.data["foo"] = "12"
	store.data["foo2"] = "bar"

	tests := []struct {
		key           string
		expectedError error
		expectedValue int
	}{
		{
			key:           "new",
			expectedError: nil,
			expectedValue: 0,
		},
		{
			key:           "foo",
			expectedError: nil,
			expectedValue: 13,
		},
		{
			key:           "foo2",
			expectedError: strconv.ErrSyntax,
			expectedValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Test INCR with key %s\n", tt.key), func(t *testing.T) {
			value, err := store.Incr(tt.key)

			if (tt.expectedError == nil && err != nil) || !errors.Is(err, tt.expectedError) {
				t.Fatalf("want err: %v, got: %v", tt.expectedError, err)
			}

			if value != tt.expectedValue {
				t.Fatalf("want %d, got: %d", tt.expectedValue, value)
			}
		})
	}
}
