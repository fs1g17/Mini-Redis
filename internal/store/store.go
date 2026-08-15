package store

import (
	"sync"
	"time"
)

type RedisStore struct {
	mu     sync.RWMutex
	data   map[string]string
	expire map[string]int64
}

func NewRedisStore() *RedisStore {
	return &RedisStore{
		data:   make(map[string]string, 512),
		expire: make(map[string]int64, 512),
	}
}

func (r *RedisStore) Set(key string, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[key] = value
}

func (r *RedisStore) Get(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now().Unix()

	// check if expired first
	if expiration, toBeExpired := r.expire[key]; toBeExpired && now >= expiration {
		// delete from both
		delete(r.data, key)
		delete(r.expire, key)
	}

	value, exists := r.data[key]
	return value, exists
}

func (r *RedisStore) Del(keys ...string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	deleted := 0
	for _, key := range keys {
		_, ok := r.data[key]
		if !ok {
			continue
		}
		delete(r.data, key)
		delete(r.expire, key)
		deleted++
	}

	return deleted
}

func (r *RedisStore) Exists(keys ...string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Unix()

	exists := 0
	for _, key := range keys {
		if expiration, expire := r.expire[key]; expire && now >= expiration {
			delete(r.data, key)
			delete(r.expire, key)
		}

		_, ok := r.data[key]
		if !ok {
			continue
		}
		exists++
	}

	return exists
}

func (r *RedisStore) Expire(key string, sec int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[key]; !ok {
		return false
	}

	r.expire[key] = time.Now().Unix() + sec

	return true
}
