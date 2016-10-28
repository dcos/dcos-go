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

package cache

import "testing"

// Smoketest that the SimpleCache works at a high level
func TestSimpleCache(t *testing.T) {
	c := SimpleCache()
	var cv interface{}
	var ok bool

	// Smoketest Set(), Get(), and Objects()
	c.Set("foo", "bar")
	cv, ok = c.Get("foo")
	if cv != "bar" {
		t.Fatalf("Expected key 'foo' to have value 'bar'. Got: %s", cv)
	}
	if ok != true {
		t.Fatalf("Expected ok to be true (but it wasn't!)")
	}
	if l := len(c.Objects()); l != 1 {
		t.Fatalf("Expected objects in the cache to be 1. Got: %d", l)
	}

	// Smoketest Delete()
	c.Delete("foo")
	if l := len(c.Objects()); l != 0 {
		t.Fatalf("Expected objects in the cache to be 0. Got: %d", l)
	}

	// Smoketest Supplant() and Size()
	someMap := make(map[string]interface{})
	testCases := []struct {
		key string
		val string
	}{
		{"foo2", "fooval2"},
		{"bar2", "barval2"},
		{"baz2", "bazval2"},
	}

	for _, tc := range testCases {
		someMap[tc.key] = tc.val
	}

	c.Supplant(someMap)
	if l := c.Size(); l != 3 {
		t.Fatalf("Expected 3 objects in the cache. Got: %d", l)
	}
}

// When deleting an object in the SimpleCache, the object should be removed completely
func TestSimpleCache_Delete(t *testing.T) {
	var c cacheImpl
	c.objects = map[string]object{}
	c.objects["foo"] = object{contents: "bar"}

	c.Delete("foo")

	if l := len(c.objects); l != 0 {
		t.Fatalf("Expected the number of cache objects to be 0. Got: %d", l)
	}

	if oc, ok := c.objects["foo"]; ok != false {
		t.Fatalf("Expected 'foo' to not be found (but it was!). Got: %s", oc)
	}
}

// When getting an object in the SimpleCache, the object's contents should be
// returned if it exists. If the requested object doesn't exist, should return nil.
func TestSimpleCache_Get(t *testing.T) {
	var c cacheImpl
	c.objects = map[string]object{}
	c.objects["foo"] = object{contents: "bar"}

	var val interface{}
	var ok bool

	val, ok = c.Get("foo")
	if val != "bar" {
		t.Fatalf("Expected value returned to be 'bar'. Got: %s", val)
	}
	if ok != true {
		t.Fatalf("Expected second assignment 'ok' to be true (but it wasn't!)")
	}

	val, ok = c.Get("someNonExistentKey")
	if val != nil {
		t.Fatalf("Expected contents of a non-existent key to be nil. Got: %v", val)
	}
	if ok != false {
		t.Fatalf("Expected second assignment 'ok' to be false (but it wasn't!)")
	}
}

// When getting all objects in the SimpleCache, all objects should be returned.
// Nothing more, nothing less.
func TestSimpleCache_Objects(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	var c cacheImpl
	c.objects = map[string]object{}

	for _, tc := range testCases {
		c.objects[tc.key] = object{contents: tc.val}
	}

	o := c.Objects()

	if l := len(o); l != 3 {
		t.Fatalf("Expected 3 items in the cache. Got: %d", l)
	}

	for _, tc := range testCases {
		if oc := o[tc.key]; oc != tc.val {
			t.Fatalf("Expected key '%s' to contain value '%s'. Got: %s", tc.key, tc.val, oc)
		}
	}
}

// When purging the SimpleCache, all objects should be deleted.
func TestSimpleCache_Purge(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	var c cacheImpl
	c.objects = map[string]object{}

	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	c.Purge()

	if l := len(c.objects); l != 0 {
		t.Fatalf("Expected 0 objects in the cache. Got: %d", l)
	}

	for _, tc := range testCases {
		if oc := c.objects[tc.key].contents; oc != nil {
			t.Fatalf("Expected to not find any objects in the cache. Got: %s", oc)
		}
	}
}

// When setting the value of an object in the SimpleCache, the value should be set
// and retrievable, and other objects in the cache should be untouched.
func TestSimpleCache_Set(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	var c cacheImpl
	c.objects = map[string]object{}

	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	if l := len(c.objects); l != 3 {
		t.Fatalf("Expected 3 objects in the cache. Got: %d", l)
	}

	for _, tc := range testCases {
		if oc := c.objects[tc.key].contents; oc != tc.val {
			t.Fatalf("Expected key '%s' to contain value '%s'. Got: %s", tc.key, tc.val, oc)
		}
	}
}

// When getting the number of objects in the SimpleCache, the correct size should be
// returned as a positive int. When adding or removing items, the new size
// should be returned.
func TestSimpleCache_Size(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	var c cacheImpl
	c.objects = map[string]object{}

	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	if l := c.Size(); l != 3 {
		t.Fatalf("Expected 3 objects in the cache. Got: %d", l)
	}
}

// When replacing all objects in the SimpleCache with a new map of cache objects,
// only the new objects should exist.
func TestSimpleCache_Supplant(t *testing.T) {
	initialTestCases := []struct {
		key string
		val string
	}{
		{"foo1", "fooval1"},
		{"bar1", "barval1"},
		{"baz1", "bazval1"},
	}
	finalTestCases := []struct {
		key string
		val string
	}{
		{"foo2", "fooval2"},
		{"bar2", "barval2"},
	}

	var c cacheImpl
	c.objects = map[string]object{}

	for _, tc := range initialTestCases {
		c.Set(tc.key, tc.val)
	}

	newMap := make(map[string]interface{})
	for _, tc := range finalTestCases {
		newMap[tc.key] = tc.val
	}

	c.Supplant(newMap)

	if l := c.Size(); l != 2 {
		t.Fatalf("Expected the cache to contain 2 objects. Got: %d", l)
	}

	for _, tc := range finalTestCases {
		if oc := c.objects[tc.key].contents; oc != tc.val {
			t.Fatalf("Expected key '%s' to contain value '%s'. Got: %s", tc.key, tc.val, oc)
		}
	}
}
