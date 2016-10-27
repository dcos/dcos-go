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
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNew(t *testing.T) {
	Convey("When creating a new instance of the cache", t, func() {
		c := New()

		Convey("Should return a new cache object", func() {
			So(c, ShouldResemble, &Cache{objects: map[string]Object{}})
		})
	})
}

func TestDelete(t *testing.T) {
	Convey("When deleting an object in the cache", t, func() {
		c := New()
		c.Set("foo", "bar")
		Convey("Should remove the object completely", func() {
			c.Delete("foo")
			_, ok := c.Get("foo")

			So(c.Size(), ShouldEqual, 0)
			So(ok, ShouldBeFalse)
		})
	})
}

func TestGet(t *testing.T) {
	Convey("When getting an object in the cache", t, func() {
		c := New()
		c.Set("foo", "bar")

		Convey("Should return the value from an object that exists", func() {
			val, ok := c.Get("foo")
			So(val, ShouldEqual, "bar")
			So(ok, ShouldBeTrue)
		})

		Convey("Should return nil for an object that doesn't exist", func() {
			val, ok := c.Get("someNonExistentKey")
			So(val, ShouldBeNil)
			So(ok, ShouldBeFalse)
		})
	})
}

func TestObjects(t *testing.T) {
	Convey("When getting all objects in the cache", t, func() {
		c := New()
		c.Set("foo", "fooval")
		c.Set("bar", "barval")
		c.Set("baz", "bazval")

		Convey("Should return all objects; nothing more, nothing less", func() {
			o := c.Objects()

			So(len(o), ShouldEqual, 3)
			So(o["foo"].Contents, ShouldEqual, "fooval")
			So(o["bar"].Contents, ShouldEqual, "barval")
			So(o["baz"].Contents, ShouldEqual, "bazval")
		})
	})
}

func TestPurge(t *testing.T) {
	Convey("When purging the cache", t, func() {
		c := New()
		c.Set("foo", "fooval")
		c.Set("bar", "barval")

		Convey("Should delete all objects", func() {
			c.Purge()
			o := c.objects

			So(len(o), ShouldEqual, 0)
			So(o["foo"].Contents, ShouldBeNil)
			So(o["bar"].Contents, ShouldBeNil)
		})
	})
}

func TestSet(t *testing.T) {
	Convey("When setting the value of an object in the cache", t, func() {
		c := New()
		c.Set("foo", "fooval")
		c.Set("bar", "barval")

		Convey("The value should be set and retrievable", func() {
			So(len(c.objects), ShouldEqual, 2)
			So(c.objects["foo"].Contents, ShouldEqual, "fooval")
		})

		Convey("Other objects in the cache should be untouched", func() {
			So(c.objects["bar"].Contents, ShouldEqual, "barval")
		})
	})
}

func TestSize(t *testing.T) {
	Convey("When getting the number of objects in the cache", t, func() {
		c := New()
		c.Set("foo", "fooval")
		c.Set("bar", "barval")
		c.Set("baz", "bazval")

		Convey("Should return the correct number of objects as a positive int", func() {
			So(c.Size(), ShouldEqual, 3)
		})

		Convey("When adding an object to the cache, the size should be larger", func() {
			c.Set("quux", "quuxval")
			So(c.Size(), ShouldEqual, 4)
			c.Delete("quux")
		})

		Convey("When removing an object from the cache, the size should be smaller", func() {
			c.Delete("foo")
			So(c.Size(), ShouldEqual, 2)
		})
	})
}

func TestSupplant(t *testing.T) {
	Convey("When replacing all objects in a cache with a new map of cache Objects", t, func() {
		c := New()
		c.Set("foo1", "fooval1")
		c.Set("bar1", "barval1")
		c.Set("baz1", "bazval1")

		Convey("Only those objects should exist", func() {
			var val interface{}
			var ok bool

			newMap := make(map[string]Object)
			newMap["foo2"] = Object{Contents: "fooval2"}
			newMap["bar2"] = Object{Contents: "barval2"}

			c.Supplant(newMap)
			fmt.Println(c.Objects())

			So(c.Size(), ShouldEqual, 2)

			val, ok = c.Get("foo2")
			So(val, ShouldEqual, "fooval2")
			So(ok, ShouldBeTrue)

			val, ok = c.Get("bar2")
			So(val, ShouldEqual, "barval2")
			So(ok, ShouldBeTrue)

			val, ok = c.Get("foo1")
			So(val, ShouldBeNil)
			So(ok, ShouldBeFalse)

			val, ok = c.Get("bar1")
			So(val, ShouldBeNil)
			So(ok, ShouldBeFalse)

			val, ok = c.Get("baz1")
			So(val, ShouldBeNil)
			So(ok, ShouldBeFalse)
		})
	})
}
