// Copyright 2016 Mesosphere, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cache is a simple, local, goroutine-safe in-memory key-value store.
package cache

import "sync"

// Cache represents the interface and available methods of the dcos-go/cache package.
// At the moment, the only implementation is SimpleCache(), although this could be
// expanded in the future to handle additional scenarios.
type Cache interface {
	Delete(string)
	Get(string) (interface{}, bool)
	Objects() map[string]interface{}
	Purge()
	Set(string, interface{})
	Size() int
	Supplant(map[string]interface{})
}

// cacheImpl represents the structure of the cache, including the cache objects
// and a single locking mechanism that is shared across a given instance, ensuring
// some level of goroutine-safety.
type cacheImpl struct {
	objects map[string]object
	mutex   sync.RWMutex
}

// object represents a single object in the cache. Although this could be represented
// as a bare interface{}, we chose to create a new object type in case we wish to add
// additional functionality or metadata in the future.
type object struct {
	contents interface{}
}

// SimpleCache creates a new, basic, in-memory cache. The caller must handle
// all Set() and Delete() operations by itself; that is to say, there is no
// concept of a maximum size or expiration on cached objects.
func SimpleCache() Cache {
	return &cacheImpl{objects: make(map[string]object)}
}

// Delete removes a single object from the cache.
func (c *cacheImpl) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.objects, key)
}

// Get returns an object from the cache.
func (c *cacheImpl) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	object, ok := c.objects[key]
	if !ok {
		return nil, false
	}

	return object.contents, true
}

// Objects returns all objects in the cache.
func (c *cacheImpl) Objects() (m map[string]interface{}) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	sz := len(c.objects)
	if sz == 0 {
		return
	}

	m = make(map[string]interface{}, sz)
	for k, v := range c.objects {
		m[k] = v.contents
	}
	return
}

// Purge removes ALL objects from the cache.
func (c *cacheImpl) Purge() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.objects = map[string]object{}
}

// Set creates an object in the cache. If the object already exists, it is overwritten.
// For bulk operations, you may wish to use Supplant() instead to avoid the overhead of
// obtaining a lock, writing data, and releasing the lock.
func (c *cacheImpl) Set(key string, val interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.objects[key] = object{contents: val}
}

// Size returns the number of objects in the cache as an integer.
func (c *cacheImpl) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.objects)
}

// Supplant replaces all objects in the cache based on a given map.
func (c *cacheImpl) Supplant(m map[string]interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	n := make(map[string]object, len(m))
	for k, v := range m {
		n[k] = object{contents: v}
	}

	c.objects = n
}
