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

import (
	"reflect"
	"testing"
)

// When creating a new instance of the cache, it should resemble &Cache
func TestNew(t *testing.T) {
	c := New()

	if !reflect.DeepEqual(c, &Cache{objects: map[string]Object{}}) {
		t.Fatalf("Expected a new cache instance to resemble &Cache. Got: %v", c)
	}
}

// When deleting an object in the cache, the object should be removed completely
func TestDelete(t *testing.T) {
	c := New()
	c.objects = map[string]Object{}
	c.objects["foo"] = Object{Contents: "bar"}

	c.Delete("foo")

	if l := len(c.objects); l != 0 {
		t.Fatalf("Expected the number of cache objects to be 0. Got: %d", l)
	}

	if oc, ok := c.objects["foo"]; ok != false {
		t.Fatalf("Expected 'foo' to not be found (but it was!). Got: %s", oc)
	}
}

// When getting an object in the cache, the object's contents should be returned
// if it exists. If the requested object doesn't exist, should return nil.
func TestGet(t *testing.T) {
	c := New()
	c.objects = map[string]Object{}
	c.objects["foo"] = Object{Contents: "bar"}

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

// When getting all objects in the cache, all objects should be returned.
// Nothing more, nothing less.
func TestObjects(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	c := New()
	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	o := c.Objects()

	if l := len(o); l != 3 {
		t.Fatalf("Expected 3 items in the cache. Got: %d", l)
	}

	for _, tc := range testCases {
		if oc := o[tc.key].Contents; oc != tc.val {
			t.Fatalf("Expected key '%s' to contain value '%s'. Got: %s", tc.key, tc.val, oc)
		}
	}
}

// When purging the cache, all objects should be deleted.
func TestPurge(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	c := New()
	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	c.Purge()

	if l := len(c.objects); l != 0 {
		t.Fatalf("Expected 0 objects in the cache. Got: %d", l)
	}

	for _, tc := range testCases {
		if oc := c.objects[tc.key].Contents; oc != nil {
			t.Fatalf("Expected to not find any objects in the cache. Got: %s", oc)
		}
	}
}

// When setting the value of an object in the cache, the value should be set
// and retrievable, and other objects in the cache should be untouched.
func TestSet(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	c := New()
	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	if l := len(c.objects); l != 3 {
		t.Fatalf("Expected 3 objects in the cache. Got: %d", l)
	}

	for _, tc := range testCases {
		if oc := c.objects[tc.key].Contents; oc != tc.val {
			t.Fatalf("Expected key '%s' to contain value '%s'. Got: %s", tc.key, tc.val, oc)
		}
	}
}

// When getting the number of objects in the cache, the correct size should be
// returned as a positive int. When adding or removing items, the new size
// should be returned.
func TestSize(t *testing.T) {
	testCases := []struct {
		key string
		val string
	}{
		{"foo", "fooval"},
		{"bar", "barval"},
		{"baz", "bazval"},
	}

	c := New()
	for _, tc := range testCases {
		c.Set(tc.key, tc.val)
	}

	if l := c.Size(); l != 3 {
		t.Fatalf("Expected 3 objects in the cache. Got: %d", l)
	}
}

// When replacing all objects in the cache with a new map of cache objects,
// only the new objects should exist.
func TestSupplant(t *testing.T) {
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

	c := New()
	for _, tc := range initialTestCases {
		c.Set(tc.key, tc.val)
	}

	newMap := make(map[string]Object)
	for _, tc := range finalTestCases {
		newMap[tc.key] = Object{Contents: tc.val}
	}

	c.Supplant(newMap)

	if l := c.Size(); l != 2 {
		t.Fatalf("Expected the cache to contain 2 objects. Got: %d", l)
	}

	for _, tc := range finalTestCases {
		if oc := c.objects[tc.key].Contents; oc != tc.val {
			t.Fatalf("Expected key '%s' to contain value '%s'. Got: %s", tc.key, tc.val, oc)
		}
	}
}
