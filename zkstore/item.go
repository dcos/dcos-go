package zkstore

import (
	"fmt"

	"github.com/pkg/errors"
)

// Item represents the data of a particular item in the store
type Item struct {
	// Ident identifies an Item in the ZK backend.
	Ident Ident

	// Data represents the bytes to be stored within the znode.
	Data []byte
}

// Validate performs validation on the Item
func (i Item) Validate() error {
	if err := i.Ident.Validate(); err != nil {
		return err
	}
	if len(i.Data) > 1024*1024 {
		return errors.New("data is greater than 1MB")
	}
	return nil
}

func (i Item) String() string {
	return fmt.Sprintf("{ident=%v data=%dB}", i.Ident, len(i.Data))
}
