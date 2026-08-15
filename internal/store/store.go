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

func (r *RedisStore) checkExpire(key string) {
	now := time.Now().Unix()

	if expiration, toBeExpired := r.expire[key]; toBeExpired && now >= expiration {
		delete(r.data, key)
		delete(r.expire, key)
	}
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

	delete(r.expire, key)
}

func (r *RedisStore) Get(key string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checkExpire(key)

	value, exists := r.data[key]
	return value, exists
}

func (r *RedisStore) Del(keys ...string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	deleted := 0
	for _, key := range keys {
		r.checkExpire(key)
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

	exists := 0
	for _, key := range keys {
		r.checkExpire(key)

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

func (r *RedisStore) TTL(key string) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checkExpire(key)

	_, keyOk := r.data[key]
	if !keyOk {
		return -2
	}

	expiration, expirationOk := r.expire[key]
	if !expirationOk {
		return -1
	}

	ttl := expiration - time.Now().Unix()
	return ttl
}
