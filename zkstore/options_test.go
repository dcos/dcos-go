package zkstore

import (
	"crypto/md5"
	"crypto/sha1"
	"testing"

	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/require"
)

func TestOptBasePath(t *testing.T) {
	require := require.New(t)
	store := &Store{}
	require.EqualError(OptBasePath("foo")(store), "basePath must start with /")
	require.NoError(OptBasePath("/foo")(store))
}

func TestOptHashBuckets(t *testing.T) {
	require := require.New(t)
	store := &Store{}
	require.EqualError(OptNumHashBuckets(0)(store), "numBuckets must be positive")
	require.EqualError(OptNumHashBuckets(-1)(store), "numBuckets must be positive")
	require.NoError(OptNumHashBuckets(1)(store))
}

func TestOptACL(t *testing.T) {
	require := require.New(t)
	store := &Store{}
	require.EqualError(OptACL(nil)(store), "ACL required")
	require.EqualError(OptACL([]zk.ACL{})(store), "ACL required")
	require.NoError(OptACL(zk.WorldACL(zk.PermAll))(store))
}

func TestOptHashProviderFunc(t *testing.T) {
	require := require.New(t)
	store := &Store{}
	require.EqualError(OptHashProviderFunc(nil)(store), "hash provider func required")
	require.NoError(OptHashProviderFunc(md5.New)(store))
	require.NoError(OptHashProviderFunc(sha1.New)(store))
}
