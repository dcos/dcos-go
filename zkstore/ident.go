package zkstore

import (
	"fmt"

	"github.com/pkg/errors"
)

// Ident specifies the location of a stored item
type Ident struct {
	// Location points to where an item lives in the store.
	Location Location

	// Version, if specified, specifies a named version of the data
	Version string

	// ZKVersion specifies the ZK version of the data.  This will be
	// used to prevent accidental overwrites.  If ZKVersion is nil,
	// then the version will not be considered for operations.
	ZKVersion *int32
}

// SetZKVersion sets the pointer of the argument as the ZK version on the Ident
func (i *Ident) SetZKVersion(version int32) {
	i.ZKVersion = &version
}

func (i Ident) String() string {
	return fmt.Sprintf("{loc=%v ver=%v zkVer=%v}", i.Location, i.Version, i.ZKVersion)
}

// Validate performs validation on the Ident
func (i Ident) Validate() error {
	if err := i.Location.Validate(); err != nil {
		return errors.Wrap(err, "invalid location")
	}
	if err := validateNamed(i.Version, false); err != nil {
		return errors.Wrap(err, "invalid version")
	}
	return nil
}

// actualZKVersion converts the *int32 into a version recognized by ZK.  ZK
// expects a -1 if the version should be ignored.
func (i Ident) actualZKVersion() int32 {
	if i.ZKVersion == nil {
		return -1
	}
	return *i.ZKVersion
}
