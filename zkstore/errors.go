package zkstore

type internalError string

func (i internalError) Error() string { return string(i) }

var _ = error(internalError("")) // sanity check

const (
	ErrIllegalOption = internalError("illegal option configuration")
	ErrHashOverflow  = internalError("hash value larger than 64 bits")

	// ErrVersionConflict is returned when a specified ZKVersion is rejected by
	// ZK when performing a mutating operation on a znode.  Clients that receive
	// this can retry by re-reading the Item and then trying again.
	ErrVersionConflict = internalError("zk version conflict")
)
