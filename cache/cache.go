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

// Package cache is a simple, local, in-memory key-value store.
package cache

import "sync"

// Object represents a single object in the cache. Although this could be represented
// as a bare interface{}, we chose to create a new Object type in case we wish to add
// additional functionality or metadata in the future.
type Object struct {
	Contents interface{}
}

// Cache represents the structure of the objects in the cache (objects)
// and a single locking mechanism (mutex) that is shared across a given instance
// of the cache.
type Cache struct {
	objects map[string]Object
	mutex   sync.RWMutex
}

// New creates a new cache and returns it to the caller. The caller is then
// responsible for creating and deleting objects.
func New() *Cache {
	return &Cache{objects: make(map[string]Object)}
}

// Delete removes a single object from the cache.
func (c *Cache) Delete(key string) {
	delete(c.objects, key)
}

// Get returns an object from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	object, ok := c.objects[key]
	if !ok {
		return nil, false
	}

	return object.Contents, true
}

// Objects returns all items in the cache as a map of objects, with a string as the key.
func (c *Cache) Objects() map[string]Object {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.objects
}

// Purge removes ALL objects from the cache.
func (c *Cache) Purge() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.objects = map[string]Object{}
}

// Set creates an object in the cache. If the object already exists, it is overwritten. For bulk operations,
// you may wish to use Supplant(map[string]cache.Object) in your code instead.
func (c *Cache) Set(key string, val interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.objects[key] = Object{Contents: val}
}

// Size returns the number of objects in the cache as an integer.
func (c *Cache) Size() int {
	return len(c.objects)
}

// Supplant replaces all objects in the cache based on a given map. This requires
// that the caller has knowledge of our public Cache and Object structures, but
// may make bulk modifications easier for some scenarios.
func (c *Cache) Supplant(m map[string]Object) {
	c.objects = m
}
