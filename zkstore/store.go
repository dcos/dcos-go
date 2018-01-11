package zkstore

import (
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

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
		if err := opt.Apply(store); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// Put stores the specified item.  If successful, an Ident will be returned
// that reflects the updated metadata for that item (specifically, the
// Version)
//
// If a client attempts to set an Item with a Version, and the current
// item (with no Version) does not exist yet, the current item will be
// created as well, with the same data as the specified Item.
func (s *Store) Put(item Item) (Ident, error) {
	err := func() error {
		if err := item.Validate(); err != nil {
			return err
		}
		identPath, err := s.identPath(item.Ident)
		if err != nil {
			return err
		}
		// shortcut: try to set it if it already exists
		stat, err := s.conn.Set(identPath, item.Data, item.Ident.actualVersion())
		switch {
		case err == zk.ErrNoNode:
			// it didn't exist, so take the more expensive path
			if stat, err = s.setFully(item); err != nil {
				return err
			}
		case err == zk.ErrBadVersion:
			return ErrVersionConflict
		case err != nil:
			return err
		case stat == nil:
			return errors.Errorf("could not stat %v", identPath)
		}
		item.Ident.Version = NewVersion(stat.Version)
		return nil
	}()
	return item.Ident, err
}

// setFully sets data for a path, creating any parents nodes as necessary.
// The stat returned will be the stat of the final created node.
func (s *Store) setFully(item Item) (stat *zk.Stat, err error) {
	err = func() error {
		identPath, err := s.identPath(item.Ident)
		if err != nil {
			return err
		}
		current := "/"
		segments := strings.Split(identPath, "/")
		for i, segment := range segments {
			isLast := i == len(segments)-1
			current = path.Join(current, segment)
			exists, _, err := s.conn.Exists(current)
			switch {
			case err != nil:
				return errors.Wrapf(err, "could not check %v", current)
			case exists:
				continue
			case isLast && item.Ident.actualVersion() >= 0:
				// specifying a new version on a non-existent node
				// is not supported
				return ErrVersionConflict
			}
			// this node does not exist. try to create it.
			var nodeData []byte

			// if the item has a variant, and its parent node does
			// not yet exist, we set the content on the parent
			// node as well as the variant node.
			isParentOfVersion := item.Ident.Variant != "" && i == len(segments)-2
			if isLast || isParentOfVersion {
				nodeData = item.Data
			}
			_, err = s.conn.Create(current, nodeData, 0, s.acls)
			if err != nil && err != zk.ErrNodeExists {
				return err
			}
		}
		stat, err = s.mustExist(identPath)
		return errors.Wrapf(err, "%v was not created", identPath)
	}()
	return
}

// Get fetches the data for a particuar item. If a particluar version is
// desired, it must be set on the ident.
func (s *Store) Get(ident Ident) (item Item, found bool, err error) {
	err = func() error {
		if err := ident.Validate(); err != nil {
			return err
		}
		identPath, err := s.identPath(ident)
		if err != nil {
			return err
		}
		data, stat, err := s.conn.Get(identPath)
		switch {
		case err == zk.ErrNoNode:
			// leave found set to false and return
			return nil
		case err != nil:
			return err
		}
		item.Ident = ident
		item.Data = data
		item.Ident.Version = NewVersion(stat.Version)
		found = true
		return nil
	}()
	return
}

// Versions fetches all of the versions for a particular item.
//
// If the item does not exist, found will be false, otherwise true.
func (s *Store) Versions(location Location) (versions []string, found bool, err error) {
	err = func() error {
		if err := location.Validate(); err != nil {
			return err
		}
		// create an ident in order to get the full path for the item node
		ident := Ident{Location: location}
		identPath, err := s.identPath(ident)
		if err != nil {
			return err
		}
		versions, _, err = s.conn.Children(identPath)
		switch {
		case err == zk.ErrNoNode:
			// leave found set to false and return
			return nil
		case err != nil:
			return err
		}
		found = true
		return nil
	}()
	return
}

// Delete deletes the identified item.  If successful, true will be returned.
// If the item does not exist, false will be returned.
func (s *Store) Delete(ident Ident) (found bool, err error) {
	if err = ident.Validate(); err != nil {
		return false, err
	}
	if ident.Variant != "" {
		return s.deleteVariant(ident)
	}
	return s.deleteItem(ident)
}

// deleteItem deletes the item and all versions within it
func (s *Store) deleteItem(ident Ident) (found bool, err error) {
	err = func() error {
		var variants []string
		variants, found, err = s.Versions(ident.Location)
		switch {
		case err != nil:
			return err
		case !found:
			// the item could not be found. leave found=false and return
			return nil
		}
		// delete each variant of the specified item
		for _, v := range variants {
			variant := ident
			variant.Variant = v
			variant.Version = Version{} // force delete it no matter the zk version
			if _, err = s.deleteVariant(variant); err != nil {
				return err
			}
		}
		// and then delete the actual parent node
		identPath, err := s.identPath(ident)
		if err != nil {
			return err
		}
		err = s.conn.Delete(identPath, ident.actualVersion())
		switch err {
		case zk.ErrNoNode:
			return nil
		case zk.ErrBadVersion:
			return ErrVersionConflict
		}
		return err
	}()
	return
}

// deleteVariant deletes only an item variant
func (s *Store) deleteVariant(ident Ident) (found bool, err error) {
	err = func() error {
		identPath, err := s.identPath(ident)
		if err != nil {
			return err
		}
		err = s.conn.Delete(identPath, ident.actualVersion())
		switch err {
		case zk.ErrNoNode:
			// perhaps someone already deleted it? leave found==false
			return nil
		case zk.ErrBadVersion:
			return ErrVersionConflict
		}
		found = true
		return err
	}()
	return
}

// List lists all of the known, latest version Locations that exist under
// the specified category.
// If the category could not be found, found will be set to false.
func (s *Store) List(category string) (locations []Location, found bool, err error) {
	err = func() error {
		if err = validateCategory(category); err != nil {
			return errors.Wrap(err, "invalid category")
		}
		bucketsPath, err := s.bucketsPath(category)
		if err != nil {
			return err
		}
		buckets, _, err := s.conn.Children(bucketsPath)
		switch {
		case err == zk.ErrNoNode:
			return nil
		case err != nil:
			return err
		}
		if len(buckets) == 0 {
			return nil
		}
		locations = make([]Location, 0, len(buckets))
		found = true
		for _, bucket := range buckets {
			children, _, err := s.conn.Children(path.Join(bucketsPath, bucket))
			switch {
			case err == zk.ErrNoNode:
				// someone else deleted it? keep going.
				continue
			case err != nil:
				return err
			}
			for _, child := range children {
				leaf := path.Base(child)
				locations = append(locations, Location{
					Category: category,
					Name:     leaf,
				})
			}
		}
		return nil
	}()
	return
}

// LocationsByName returns a sort function helper that may be passed to sort.Slice in order to sort
// a slice of Location structs.
func LocationsByName(locations []Location) (interface{}, func(_, _ int) bool) {
	return locations, func(i, j int) bool {
		return locations[i].Name < locations[j].Name
	}
}

// Close shuts down the Store
func (s *Store) Close() error {
	return s.closeFunc()
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
	return
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
		ident.Variant,
	), nil
}

// bucketsPath returns the full path of the buckets znode for a given category.
// an error will be returned if the category ends with the name of the buckets
// znode name.
func (s *Store) bucketsPath(category string) (string, error) {
	leaf := path.Base(path.Clean(category))
	if leaf == s.bucketsZnodeName {
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
	hash, err := s.hashProviderFunc(name)
	if err != nil {
		return 0, err
	}
	return hashBytesToBucket(s.hashBuckets, hash), nil
}
