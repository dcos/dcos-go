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
