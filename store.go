package main

import "sync"

type Store interface {
	Set(key string, value string)
	Get(key string) (string, bool)
	Del(key ...string) int
	Exists(key ...string) int
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
		deleted++
	}

	return deleted
}

func (r *RedisStore) Exists(keys ...string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exists := 0
	for _, key := range keys {
		_, ok := r.data[key]
		if !ok {
			continue
		}
		exists++
	}

	return exists
}
