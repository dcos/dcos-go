package zkstore

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"hash"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

// IStore is the interface to which Store confirms. Documentation for these
// methods lives on the concrete Store type, as the NewStore method returns
// a concrete type, not this interface.  Clients may choose to use the IStore
// interface if they wish.
type IStore interface {
	Put(item Item) (Ident, error)
	Get(ident Ident) (item Item, found bool, err error)
	List(category string) (locations []Location, found bool, err error)
	Versions(location Location) (versions []string, found bool, err error)
	Delete(ident Ident) (found bool, err error)
	Close() error
}

// ensure that Store confirms to the IStore interface.
var _ IStore = &Store{}

// HashProviderFunc is a factory for hashers.
type HashProviderFunc func() hash.Hash

// Store exposes an API for performing CRUD operations against a backing
// ZK cluster.
type Store struct {
	conn             *zk.Conn                  // the zk connection
	basePath         string                    // the base path to use for any znodes
	bucketsZnodeName string                    // the name of the znode folder
	acls             []zk.ACL                  // the ACLs to use for any created nodes
	hashBuckets      int                       // the number of hash buckets
	bucketFunc       func(string) (int, error) // converts a name into a bucket number
	hashProviderFunc HashProviderFunc          // converts a name into a hash
	closeFunc        func() error              // closes zk resources
}

const (
	// DefaultNumHashBuckets is the number of buckets that will be used to
	// spread out items living within a category by placing children of the
	// category into numerically named buckets.  Clients may choose to override
	// this by specifying OptNumHashBuckets when building the Store.
	DefaultNumHashBuckets = 256

	// DefaultBucketsZnodeName is the default name of parent znode that will
	// store the buckets for a particular category.  This is necessary to
	// allow categories like "foo" and "foo/bar", since we enforce that
	// category names cannot end in this name.  Clients may choose to override
	// this default by specifying OptBucketsZnodeName when building the
	// Store.
	DefaultBucketsZnodeName = "buckets"
)

var (
	// ErrVersionConflict is returned when a specified ZKVersion is rejected by
	// ZK when performing a mutating operation on a znode.  Clients that receive
	// this can retry by re-reading the Item and then trying again.
	ErrVersionConflict = errors.New("zk version conflict")

	// DefaultHashProviderFunc is the default provider of hash functions
	// unless overriden with OptHashProviderFunc when building the Store.
	DefaultHashProviderFunc HashProviderFunc = md5.New

	// DefaultZKACL is the default ACL set that will be used when creating
	// nodes in ZK unless overridden with OptACL when building the Store.
	DefaultZKACL = zk.WorldACL(zk.PermAll)
)

// NewStore creates a new Store that is ready for use.
func NewStore(connector Connector, opts ...StoreOpt) (*Store, error) {
	conn, err := connector.Connect()
	if err != nil {
		return nil, err
	}
	store := &Store{
		conn:             conn,
		closeFunc:        connector.Close,
		bucketsZnodeName: DefaultBucketsZnodeName,
		hashBuckets:      DefaultNumHashBuckets,
		acls:             DefaultZKACL,
		hashProviderFunc: DefaultHashProviderFunc,
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, errors.New("nil opt")
		}
		if err := opt(store); err != nil {
			return nil, err
		}
	}
	return store, nil
}
// mustExist checks whether or not the path exists, and returns an error
// if it could not be verified to exist.
func (s *Store) mustExist(path string) (stat *zk.Stat, err error) {
	err = func() error {
		var exists bool
		exists, stat, err = s.conn.Exists(path)
		switch {
		case err != nil:
			return err
		case !exists:
			return errors.Errorf("%v did not exist ", path)
		case stat == nil:
			return errors.Errorf("got nil stat for path %v", path)
		}
		return nil
	}()
	return stat, err
}

// identPath returns the full path of the item pointed to by the Ident
func (s *Store) identPath(ident Ident) (string, error) {
	bucket, err := s.bucketFor(ident.Location.Name)
	if err != nil {
		return "", err
	}
	bucketsPath, err := s.bucketsPath(ident.Location.Category)
	if err != nil {
		return "", err
	}
	return path.Join(
		bucketsPath,
		strconv.Itoa(bucket),
		ident.Location.Name,
		ident.Version,
	), nil
}

// bucketsPath returns the full path of the buckets znode for a given category.
// an error will be returned if the category ends with the name of the buckets
// znode name.
func (s *Store) bucketsPath(category string) (string, error) {
	segments := strings.Split(path.Clean(category), "/")
	if segments[len(segments)-1] == s.bucketsZnodeName {
		return "", errors.New("category may not end with the buckets znode name")
	}
	return path.Join(
		"/",
		s.basePath,
		category,
		s.bucketsZnodeName,
	), nil
}

// bucketFor returns the bucket number for a particular name
func (s *Store) bucketFor(name string) (int, error) {
	if s.bucketFunc != nil {
		return s.bucketFunc(name)
	}
	hasher := s.hashProviderFunc()
	_, err := hasher.Write([]byte(name))
	if err != nil {
		return 0, err
	}
	hash := hasher.Sum(nil)
	return s.hashBytesToBucket(hash)
}

// hashBytesToBucket produces a positive int from a hashed byte slice
func (s *Store) hashBytesToBucket(hash []byte) (int, error) {
	var res int64
	reader := bytes.NewReader(hash[len(hash)-8:])
	if err := binary.Read(reader, binary.LittleEndian, &res); err != nil {
		return 0, err
	}
	mod := int(res) % s.hashBuckets
	if mod < 0 {
		mod = mod * -1
	}
	return mod, nil
}
