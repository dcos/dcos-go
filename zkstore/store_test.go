package zkstore

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha512"
	"fmt"
	"io"
	"testing"

	"github.com/dcos/dcos-go/testutils"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/require"
)

// TODO: perhaps use a _buckets folder for the buckets. that would allow
// any sort of category to be used as long as we don't allow _ as the first
// character for a named item?

func TestExpectedBehavior(t *testing.T) {
	store, _, teardown := newStoreTest(t, OptBasePath("/storage"))
	defer teardown()
	require := require.New(t)

	locations, found, err := store.List("widgets")
	require.NoError(err)
	require.False(found)
	require.Nil(locations)

	item1 := Item{
		Ident: Ident{Location: Location{Category: "widgets", Name: "item1"}},
		Data:  []byte("item1"),
	}
	item2 := Item{
		Ident: Ident{Location: Location{Category: "widgets", Name: "item2"}},
		Data:  []byte("item2"),
	}

	_, err = store.Put(item1)
	require.NoError(err)
	_, err = store.Put(item2)
	require.NoError(err)

	locations, found, err = store.List("widgets")
	require.NoError(err)
	require.True(found)
	require.EqualValues([]Location{
		{Category: "widgets", Name: "item1"},
		{Category: "widgets", Name: "item2"},
	}, locations)

	// set a version on item1
	_, err = store.Put(Item{
		Ident: Ident{Location: Location{Category: "widgets", Name: "item1"}, Version: "v2"},
		Data:  []byte("item1v2"),
	})
	require.NoError(err)

	// locations should not include versions
	locations, found, err = store.List("widgets")
	require.NoError(err)
	require.True(found)
	require.EqualValues([]Location{
		{Category: "widgets", Name: "item1"},
		{Category: "widgets", Name: "item2"},
	}, locations)

	// we can get the versions for an item if we ask for it specifically
	versions, found, err := store.Versions(Location{Category: "widgets", Name: "item1"})
	require.NoError(err)
	require.True(found)
	require.EqualValues([]string{"v2"}, versions)

	// fetch a particular version
	item, found, err := store.Get(Ident{Location: Location{Category: "widgets", Name: "item1"}, Version: "v2"})
	require.NoError(err)
	require.True(found)
	require.EqualValues("item1v2", string(item.Data))

	// we should still be able to fetch the first one
	item, found, err = store.Get(Ident{Location: Location{Category: "widgets", Name: "item1"}})
	require.NoError(err)
	require.True(found)
	require.EqualValues("item1", string(item.Data))

	// try to delete a node that doesn't exist
	found, err = store.Delete(Ident{Location: Location{Category: "widgets", Name: "item-none"}})
	require.NoError(err)
	require.False(found)

	// try to delete a version that doesn't exist
	found, err = store.Delete(Ident{Location: Location{Category: "widgets", Name: "item-none"}, Version: "v2"})
	require.NoError(err)
	require.False(found)

	// delete the node without any children
	found, err = store.Delete(Ident{Location: Location{Category: "widgets", Name: "item2"}})
	require.NoError(err)
	require.True(found)

	// try to query the node after you deleted it
	found, err = store.Delete(Ident{Location: Location{Category: "widgets", Name: "item2"}})
	require.NoError(err)
	require.False(found)

	// query the locations for the widgets category. we should only have one now.
	locations, found, err = store.List("widgets")
	require.NoError(err)
	require.True(found)
	require.EqualValues([]Location{{Category: "widgets", Name: "item1"}}, locations)

	// update the remaining item
	_, err = store.Put(Item{
		Ident: Ident{Location: Location{Category: "widgets", Name: "item1"}},
		Data:  []byte("item1updated"),
	})
	require.NoError(err)

	// verify the overwrite
	item, found, err = store.Get(Ident{Location: Location{Category: "widgets", Name: "item1"}})
	require.NoError(err)
	require.True(found)
	require.Equal("item1updated", string(item.Data))

	// delete the first item and all of its versions
	found, err = store.Delete(Ident{Location: Location{Category: "widgets", Name: "item1"}})
	require.NoError(err)
	require.True(found)

	// verify that it was deleted
	item, found, err = store.Get(Ident{Location: Location{Category: "widgets", Name: "item1"}})
	require.NoError(err)
	require.False(found)

	// verify that its version was also deleted
	item, found, err = store.Get(Ident{Location: Location{Category: "widgets", Name: "item1"}, Version: "v2"})
	require.NoError(err)
	require.False(found)
}

func TestVersionParentNodeDataIsNotSetIfItAlreadyExists(t *testing.T) {
	store, conn, teardown := newStoreTest(t, fixedBucketFunc(42), OptBasePath("/storage"))
	defer teardown()
	require := require.New(t)

	_, err := store.Put(Item{
		Ident: Ident{
			Location: Location{
				Category: "/widgets",
				Name:     "foo",
			},
		},
		Data: []byte("parent"),
	})
	require.NoError(err)

	_, err = store.Put(Item{
		Ident: Ident{
			Location: Location{
				Category: "/widgets",
				Name:     "foo",
			},
			Version: "my-version",
		},
		Data: []byte("child"),
	})
	require.NoError(err)

	// verify that the parent node exists with the data
	data, stat, err := conn.Get("/storage/widgets/buckets/42/foo")
	require.NoError(err)
	require.NotNil(stat)
	require.Equal("parent", string(data))

	// verify that the version node exists with the data
	data, stat, err = conn.Get("/storage/widgets/buckets/42/foo/my-version")
	require.NoError(err)
	require.NotNil(stat)
	require.Equal("child", string(data))
}

func TestVersionParentNodeIsSetIfItDoesNotExist(t *testing.T) {
	store, conn, teardown := newStoreTest(t, fixedBucketFunc(42), OptBasePath("/storage"))
	defer teardown()
	require := require.New(t)

	_, err := store.Put(Item{
		Ident: Ident{
			Location: Location{
				Category: "/widgets",
				Name:     "foo",
			},
			Version: "my-version",
		},
		Data: []byte("hello"),
	})
	require.NoError(err)

	// verify that the parent node exists with the data
	data, stat, err := conn.Get("/storage/widgets/buckets/42/foo")
	require.NoError(err)
	require.NotNil(stat)
	require.Equal("hello", string(data))

	// verify that the version node exists with the data
	data, stat, err = conn.Get("/storage/widgets/buckets/42/foo/my-version")
	require.NoError(err)
	require.NotNil(stat)
	require.Equal("hello", string(data))
}

// If the ZKVersion is set on an Ident, we need to ensure that the underlying
// store respects that and rejects operations for which a version was
// different from that which was specified.
func TestZKVersionsRespected(t *testing.T) {
	store, _, teardown := newStoreTest(t, fixedBucketFunc(42), OptBasePath("/storage"))
	defer teardown()
	require := require.New(t)

	newItem := func() Item {
		return Item{
			Ident: Ident{
				Location: Location{
					Category: "/widgets",
					Name:     "foo",
				},
				Version: "my-version",
			},
			Data: []byte("hello"),
		}
	}

	// this should fail because the item does not exist yet
	item := newItem()
	item.Ident.SetZKVersion(42) // invalid
	ident, err := store.Put(item)
	require.Equal(ErrVersionConflict, err)

	item = newItem()
	ident, err = store.Put(item)
	require.NoError(err)
	require.EqualValues(0, *ident.ZKVersion)

	// put the same item again, verify its zkVersion==1
	item = newItem()
	ident, err = store.Put(item)
	require.NoError(err)
	require.EqualValues(1, *ident.ZKVersion)

	// put the same item, but set it to use zkversion=1
	item = newItem()
	item.Ident.SetZKVersion(1)
	ident, err = store.Put(item)
	require.NoError(err)
	require.EqualValues(2, *ident.ZKVersion)

	// put the same item, but set a previous version
	item = newItem()
	item.Ident.SetZKVersion(1)
	ident, err = store.Put(item)
	require.EqualValues(ErrVersionConflict, err)

	// at this point, our stored item is still at version 2
	// we should get a conflict if we try to delete version 1
	item = newItem()
	item.Ident.SetZKVersion(1)
	_, err = store.Delete(item.Ident)
	require.EqualValues(ErrVersionConflict, err)
}

func TestZKVersionIsIncrementedOnPut(t *testing.T) {
	store, _, teardown := newStoreTest(t, fixedBucketFunc(42), OptBasePath("/storage"))
	defer teardown()
	require := require.New(t)

	// test zk versions for item nodes
	for i := 0; i < 5; i++ {
		ident, err := store.Put(Item{
			Ident: Ident{
				Location: Location{
					Category: "/widgets",
					Name:     "foo",
				},
			},
			Data: []byte("hello"),
		})
		require.NoError(err)
		copy := ident
		copy.SetZKVersion(int32(i))
		require.EqualValues(copy, ident)
	}

	// test zk versions for version nodes
	for i := 0; i < 5; i++ {
		ident, err := store.Put(Item{
			Ident: Ident{
				Location: Location{
					Category: "/widgets",
					Name:     "foo",
				},
				Version: "my-version",
			},
			Data: []byte("hello"),
		})
		require.NoError(err)
		copy := ident
		copy.SetZKVersion(int32(i))
		require.EqualValues(copy, ident)
	}
}

func TestListLocations(t *testing.T) {
	store, _, teardown := newStoreTest(t, fixedBucketFunc(42), OptBasePath("/storage"))
	defer teardown()
	require := require.New(t)

	_, err := store.Put(Item{
		Ident: Ident{
			Location: Location{Category: "widgets/2017", Name: "foo"},
			Version:  "my-version",
		},
		Data: []byte("hello"),
	})
	require.NoError(err)
	_, err = store.Put(Item{
		Ident: Ident{
			Location: Location{Category: "widgets/2017", Name: "bar"},
			Version:  "my-version",
		},
		Data: []byte("hello"),
	})
	require.NoError(err)

	locations, found, err := store.List("/widgets/2017")
	require.NoError(err)
	require.True(found)
	require.EqualValues([]Location{
		{Name: "bar", Category: "/widgets/2017"},
		{Name: "foo", Category: "/widgets/2017"},
	}, locations)
	require.Len(locations, 2)
}

// ensure a reasonable distribution of buckets for a range of hash functions.
//
// NB: i could not get the fnv hash to pass this test
func TestBucketDistribution(t *testing.T) {
	type hashTest struct {
		name             string
		hashProviderFunc HashProviderFunc
	}
	for _, test := range []hashTest{
		{"md5", md5.New},
		{"sha", sha1.New},
		{"sha512", sha512.New},
	} {
		require := require.New(t)
		numBuckets := 1024
		numNames := 1024 * 1024
		s, err := NewStore(noConn(), OptNumHashBuckets(numBuckets), OptHashProviderFunc(test.hashProviderFunc))
		require.NoError(err)
		hits := make(map[int]int)
		for i := 0; i < numBuckets; i++ {
			hits[i] = 0
		}
		for i := 0; i < numNames; i++ {
			b, err := s.bucketFor(fmt.Sprintf("name-%d", i))
			require.NoError(err)
			if b < 0 {
				t.Fatalf("hashFunc=%v got negative bucket value", test.name)
			}
			hits[b]++
		}
		for b, num := range hits {
			if num == 0 {
				t.Fatalf("hashFunc=%v bucket %v did not have any hits", test.name, b)
			}
		}
	}
}

func TestHashProducesSameValues(t *testing.T) {
	require := require.New(t)
	for i := 0; i < 1024; i++ {
		s, err := NewStore(noConn(), OptNumHashBuckets(64))
		require.NoError(err)
		name := fmt.Sprintf("name-%d", i)
		b1, err := s.bucketFor(name)
		require.NoError(err)
		b2, err := s.bucketFor(name)
		require.NoError(err)
		require.Equal(b1, b2)
	}
}

// TestIdentPath ensures that our path generation routines are working as
// expected.
func TestIdentPath(t *testing.T) {
	type testCase struct {
		ident    Ident
		basePath string
		path     string
		errMsg   string
	}
	for _, test := range []testCase{
		{
			ident:  Ident{Location: Location{Name: "my-name", Category: "buckets"}},
			errMsg: "category may not end with the buckets znode name",
		},
		{
			ident:  Ident{Location: Location{Name: "my-name", Category: "/buckets"}},
			errMsg: "category may not end with the buckets znode name",
		},
		{
			ident:  Ident{Location: Location{Name: "my-name", Category: "/buckets/"}},
			errMsg: "category may not end with the buckets znode name",
		},
		{
			ident:  Ident{Location: Location{Name: "my-name", Category: "foo/buckets"}},
			errMsg: "category may not end with the buckets znode name",
		},
		{
			ident:  Ident{Location: Location{Name: "my-name", Category: "/foo/buckets/"}},
			errMsg: "category may not end with the buckets znode name",
		},
		{
			ident: Ident{Location: Location{Name: "my-name", Category: "widgets"}},
			path:  "/widgets/buckets/42/my-name",
		},
		{
			ident: Ident{Location: Location{Name: "my-name", Category: "widgets/2017"}},
			path:  "/widgets/2017/buckets/42/my-name",
		},
		{
			basePath: "/storage",
			ident:    Ident{Location: Location{Name: "my-name", Category: "widgets"}},
			path:     "/storage/widgets/buckets/42/my-name",
		},
		{
			basePath: "/storage/things",
			ident:    Ident{Location: Location{Name: "my-name", Category: "widgets"}},
			path:     "/storage/things/widgets/buckets/42/my-name",
		},
		{
			basePath: "/storage/things",
			ident:    Ident{Location: Location{Name: "my-name", Category: "widgets/2017"}},
			path:     "/storage/things/widgets/2017/buckets/42/my-name",
		},
	} {
		store, err := NewStore(noConn(), OptBasePath(test.basePath), fixedBucketFunc(42))
		if err != nil {
			t.Fatalf("%#v produced err:%v", test, err)
		}
		identPath, err := store.identPath(test.ident)
		if identPath != test.path || errMsg(err) != test.errMsg {
			t.Fatalf("%#v produced path:%v err:%v", test, identPath, err)
		}
	}
}

func newStoreTest(t *testing.T, storeOpts ...StoreOpt) (store *Store, zkConn *zk.Conn, teardown func()) {
	zkCtl, err := testutils.StartZookeeper()
	if err != nil {
		t.Fatal(err)
	}
	connector := NewConnection([]string{zkCtl.Addr()}, ConnectionOpts{})
	conn, err := connector.Connect()
	if err != nil {
		t.Fatal(err)
	}
	store, err = NewStore(ExistingConnection(conn), storeOpts...)
	if err != nil {
		t.Fatal(err)
	}
	return store, conn, func() {
		closePanic(store)
		conn.Close()
		zkCtl.TeardownPanic()
	}
}

func closePanic(closer io.Closer) {
	if err := closer.Close(); err != nil {
		panic(err)
	}
}

// noConn returns a connector that does nothing, for tests on the Store which
// do not require zookeeper.
func noConn() Connector {
	return ExistingConnection(nil)
}

// fixedBucketFunc configures a store to always use a single bucket number. This
// is used to make verification easy in tests.
func fixedBucketFunc(bucket int) StoreOpt {
	return func(store *Store) error {
		store.bucketFunc = func(name string) (int, error) {
			return bucket, nil
		}
		return nil
	}
}
