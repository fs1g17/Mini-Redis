package main

import "sync"

type Store interface {
	Set(key string, value string) error
	Get(key string) (*string, error)
	Del(key string) error
	Exists(key string) (bool, error)
}

type RedisStore struct {
	mu   sync.Mutex
	data map[string]string
}

func NewRedisStore() *RedisStore {
	return &RedisStore{
		data: make(map[string]string, 512),
	}
}

func (r *RedisStore) Set(key string, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[key] = value
	return nil
}

func (r *RedisStore) Get(key string) (*string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	value, exists := r.data[key]
	if !exists {
		return nil, nil
	}

	return &value, nil
}

func (r *RedisStore) Del(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.data, key)

	return nil
}

func (r *RedisStore) Exists(key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.data[key]
	return ok, nil
}
