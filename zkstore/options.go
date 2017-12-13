package zkstore

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

// StoreOpt allows a Store to be configured.
type StoreOpt func(store *Store) error

// OptBasePath specifies a root path that will be prepended to all paths written to
// or read from.
func OptBasePath(basePath string) StoreOpt {
	return func(store *Store) error {
		if basePath == "" {
			return nil
		}
		if !strings.HasPrefix(basePath, "/") {
			return errors.New("basePath must start with /")
		}
		store.basePath = basePath
		return nil
	}
}

// OptNumHashBuckets specifies the number of hash buckets that will be created under
// a store path for each content type when data is being written or read.
//
// If this value is changed after data is written, previously written data may
// not be able to be found later.
func OptNumHashBuckets(numBuckets int) StoreOpt {
	return func(store *Store) error {
		if numBuckets <= 0 {
			return errors.New("numBuckets must be positive")
		}
		store.hashBuckets = numBuckets
		return nil
	}
}

// OptACL configures the store to use a particular ACL when creating nodes.
func OptACL(acl []zk.ACL) StoreOpt {
	return func(store *Store) error {
		if len(acl) == 0 {
			return errors.New("ACL required")
		}
		store.acls = acl
		return nil
	}
}

// OptHashProviderFunc allows the client to configure which hasher to use to map
// item names to buckets.
func OptHashProviderFunc(hashProviderFunc HashProviderFunc) StoreOpt {
	return func(store *Store) error {
		if hashProviderFunc == nil {
			return errors.New("hash provider func required")
		}
		store.hashProviderFunc = hashProviderFunc
		return nil
	}
}

// OptBucketsZnodeName allows the client to configure the znode name that will
// contain the numerically-named bucket nodes.
func OptBucketsZnodeName(name string) StoreOpt {
	return func(store *Store) error {
		if err := validateNamed(name, true); err != nil {
			return errors.Wrap(err, "invalid buckets znode name")
		}
		store.bucketsZnodeName = name
		return nil
	}
}
