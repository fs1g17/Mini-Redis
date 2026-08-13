package main

import "sync"

type Store interface {
	Set(key string, value string)
	Get(key string) (string, bool)
	Del(key string)
	Exists(key string) bool
}

type RedisStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewRedisStore() *RedisStore {
	return &RedisStore{
		data: make(map[string]string, 512),
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

	value, exists := r.data[key]
	return value, exists
}

func (r *RedisStore) Del(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.data, key)
}

func (r *RedisStore) Exists(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.data[key]
	return ok
}
