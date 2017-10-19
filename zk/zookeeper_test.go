package zk

import (
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	client, container, err := createClient(Config{})
	require.Nil(t, err)
	defer client.Close()
	defer container.Kill()
	defer container.Remove()

	err = client.Create("/zeenode", []byte("value"), nil)
	require.Nil(t, err)

	err = client.Create("/zeenode", []byte("value"), nil)
	require.Equal(t, ErrZnodeAlreadyExists, err)
}

func TestCreateEphemeral(t *testing.T) {
	client, container, err := createClient(Config{})
	require.Nil(t, err)
	defer container.Kill()
	defer container.Remove()

	err = client.CreateEphemeral("/zeenode", []byte("value"), nil)
	require.Nil(t, err)

	err = client.Create("/zeenode", []byte("value"), nil)
	require.Equal(t, ErrZnodeAlreadyExists, err)

	// When closing the connection the znode should disappear.
	client.Close()
	client, err = New(Config{
		Addr: container.Addr() + ":2181",
	})
	require.Nil(t, err)
	defer client.Close()

	err = client.CreateEphemeral("/zeenode", []byte("value"), nil)
	require.Nil(t, err)
}

func TestExists(t *testing.T) {
	client, container, err := createClient(Config{})
	require.Nil(t, err)
	defer client.Close()
	defer container.Kill()
	defer container.Remove()

	exists, err := client.Exists("/zeenode")
	require.Equal(t, false, exists)

	err = client.Create("/zeenode", []byte("value"), nil)
	require.Nil(t, err)

	exists, err = client.Exists("/zeenode")
	require.Equal(t, true, exists)
}

func TestGet(t *testing.T) {
	client, container, err := createClient(Config{})
	require.Nil(t, err)
	defer client.Close()
	defer container.Kill()
	defer container.Remove()

	data, err := client.Get("/zeenode")
	require.Nil(t, data)
	require.Equal(t, ErrZnodeDoesNotExist, err)

	err = client.Create("/zeenode", []byte("value"), nil)
	require.Nil(t, err)

	data, err = client.Get("/zeenode")
	require.Nil(t, err)
	require.Equal(t, []byte("value"), data)
}

func TestDelete(t *testing.T) {
	client, container, err := createClient(Config{})
	require.Nil(t, err)
	defer client.Close()
	defer container.Kill()
	defer container.Remove()

	err = client.Delete("/zeenode")
	require.Equal(t, ErrZnodeDoesNotExist, err)

	err = client.Create("/zeenode", []byte("value"), nil)
	require.Nil(t, err)

	// Just to increment the znode data version, subsequent delete should still succeed.
	err = client.Put("/zeenode", []byte("value"))
	require.Nil(t, err)

	err = client.Delete("/zeenode")
	require.Nil(t, err)

	exists, err := client.Exists("/zeenode")
	require.Nil(t, err)
	require.Equal(t, false, exists)
}

func TestCreateAllAndChildren(t *testing.T) {
	client, container, err := createClient(Config{})
	require.Nil(t, err)
	defer client.Close()
	defer container.Kill()
	defer container.Remove()

	err = client.CreateAll("/bookkeeper/book", []byte("Once upon a time..."), nil)
	require.Nil(t, err)

	// parent node shouldn't hold any value
	parent, err := client.Get("/bookkeeper")
	require.Nil(t, err)
	require.Equal(t, []byte(nil), parent)

	child, err := client.Children("/bookkeeper")
	require.Nil(t, err)
	require.Equal(t, []string{"book"}, child)
}

func TestParseSchemeAuth(t *testing.T) {
	schemeAuth, err := ParseSchemeAuth("digest:admin:bjkZ9W+M82HUZ9xb8/Oy4cmJGfg=")
	require.Nil(t, err)
	require.Equal(t, "digest", schemeAuth.Scheme)
	require.Equal(t, "admin:bjkZ9W+M82HUZ9xb8/Oy4cmJGfg=", schemeAuth.Auth)

	_, err = ParseSchemeAuth("digest:")
	require.NotNil(t, err)
}

func TestParseSchemeID(t *testing.T) {
	schemeID, err := ParseSchemeID("auth:")
	require.Nil(t, err)
	require.Equal(t, "auth", schemeID.Scheme)
	require.Equal(t, "", schemeID.ID)

	_, err = ParseSchemeAuth(":")
	require.NotNil(t, err)
}

func TestDefaultACL(t *testing.T) {
	config := Config{
		DefaultID: &SchemeID{
			Scheme: "auth",
			ID:     "",
		},
	}
	client, container, err := createClient(config)
	require.Nil(t, err)
	defer client.Close()
	defer container.Kill()
	defer container.Remove()

	defaultACL := client.DefaultACL(PermRead)
	require.Equal(t, 1, len(defaultACL))
	require.Equal(t, PermRead, defaultACL[0].Perms)
	require.Equal(t, "auth", defaultACL[0].Scheme)
	require.Equal(t, "", defaultACL[0].ID)
}

func TestAuthACL(t *testing.T) {
	authACL := AuthACL(PermWrite)
	require.Equal(t, 1, len(authACL))
	require.Equal(t, PermWrite, authACL[0].Perms)
	require.Equal(t, "auth", authACL[0].Scheme)
	require.Equal(t, "", authACL[0].ID)
}

func TestWorldACL(t *testing.T) {
	worldACL := WorldACL(PermRead | PermWrite)
	require.Equal(t, 1, len(worldACL))
	require.Equal(t, PermRead|PermWrite, worldACL[0].Perms)
	require.Equal(t, "world", worldACL[0].Scheme)
	require.Equal(t, "anyone", worldACL[0].ID)
}

func TestDigestACL(t *testing.T) {
	digestACL := DigestACL(PermRead, "admin", "pass")
	require.Equal(t, 1, len(digestACL))
	require.Equal(t, PermRead, digestACL[0].Perms)
	require.Equal(t, "digest", digestACL[0].Scheme)
	require.Equal(t, "admin:DlzbIo9gQYKF6PVIhs7ZYzUii2w=", digestACL[0].ID)
}

func createClient(config Config) (*Client, *container, error) {
	container, err := newZkContainer()
	if err != nil {
		return nil, nil, err
	}

	config.Addr = container.Addr() + ":2181"

	// give ~10 seconds to zookeeper to become reachable
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", config.Addr)
		time.Sleep(200 * time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
	}

	client, err := New(config)
	if err != nil {
		container.Kill()
		container.Remove()
		return nil, nil, err
	}
	return client, container, nil
}

type container struct {
	id   string
	addr string
}

func (c *container) Addr() string {
	return c.addr
}

func (c *container) Kill() error {
	return exec.Command("docker", "kill", c.id).Run()
}

// TODO: Instead of this, the --rm option could be added when running the container,
// this is however not supported with the old docker version we have in Jenkins (1.12).
func (c *container) Remove() error {
	return exec.Command("docker", "rm", c.id).Run()
}

func newZkContainer() (*container, error) {
	// run container and get its id
	out, err := exec.Command("docker", "run", "-d", "zookeeper:3.4").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(out))

	// get container ip
	out, err = exec.Command("docker", "inspect", "--format={{.NetworkSettings.IPAddress}}", id).Output()
	if err != nil {
		return nil, err
	}
	ip := strings.TrimSpace(string(out))

	return &container{
		id:   id,
		addr: ip,
	}, nil
}
