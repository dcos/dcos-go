// Package zk provides functions for getting and putting data in a znode.
package zk

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

// DefaultAddr is the default address for zookeeper.
const DefaultAddr = "127.0.0.1:2181"

// Permissions bits to be used for ACLs
const (
	PermRead   = int32(zk.PermRead)
	PermWrite  = int32(zk.PermWrite)
	PermCreate = int32(zk.PermCreate)
	PermDelete = int32(zk.PermDelete)
	PermAdmin  = int32(zk.PermAdmin)
	PermAll    = int32(zk.PermAll)
)

var (
	// ErrZnodeDoesNotExist is returned if the requested znode does not exist.
	ErrZnodeDoesNotExist = zk.ErrNoNode
	// ErrZnodeAlreadyExists is returned if a given znode already exists.
	ErrZnodeAlreadyExists = zk.ErrNodeExists
)

// ACL defines permissions, a scheme and an ID.
type ACL struct {
	Perms  int32
	Scheme string
	ID     string
}

// Logger is used for debugging purposes
type Logger interface {
	Printf(string, ...interface{})
}

type discardLogger struct{}

func (*discardLogger) Printf(format string, a ...interface{}) {
}

// Config defines the default parameters for zookeeper setup.
type Config struct {
	// Address for zookeeper, an empty string will use DefaultAddr
	Addr string

	// If specified, all paths for the Client will be prefixed by this
	BasePath string

	// Authentication info for the connection
	Auth *SchemeAuth

	// Default ID to be used for Client.DefaultACL function
	DefaultID *SchemeID

	// A logger for the client
	Logger Logger
}

// SchemeAuth composes scheme and auth
type SchemeAuth struct {
	Scheme string
	Auth   string
}

// ParseSchemeAuth parses input such as "scheme:auth"
func ParseSchemeAuth(schemeAuth string) (*SchemeAuth, error) {
	splits := strings.SplitN(schemeAuth, ":", 2)
	if len(splits) != 2 || splits[0] == "" || splits[1] == "" {
		return nil, fmt.Errorf("schemeAuth expected format is 'schema:auth'")
	}

	return &SchemeAuth{
		Scheme: splits[0],
		Auth:   splits[1],
	}, nil

}

// SchemeID composes a scheme and id
type SchemeID struct {
	Scheme string
	ID     string
}

// ParseSchemeID parses input such as "scheme:id"
func ParseSchemeID(schemeID string) (*SchemeID, error) {
	splits := strings.SplitN(schemeID, ":", 2)
	if len(splits) != 2 || splits[0] == "" {
		return nil, fmt.Errorf("schemeID expected format is 'schema:id'")
	}

	return &SchemeID{
		Scheme: splits[0],
		ID:     splits[1],
	}, nil
}

// Client exports the main API that users of this package will use
type Client struct {
	conn   *zk.Conn
	acl    []ACL
	config Config
}

// New creates a new zookeeper Client.
func New(config Config) (*Client, error) {
	// ensure path is suffixed and prefixed (zk requires prefix /)
	if !strings.HasSuffix(config.BasePath, "/") {
		config.BasePath += "/"
	}
	if !strings.HasPrefix(config.BasePath, "/") {
		config.BasePath = "/" + config.BasePath
	}

	if config.DefaultID == nil {
		config.DefaultID = &SchemeID{
			Scheme: "world",
			ID:     "anyone",
		}
	}

	// create the default ACL
	acl := []ACL{
		{
			Perms:  PermAll,
			Scheme: config.DefaultID.Scheme,
			ID:     config.DefaultID.ID,
		},
	}

	if config.Logger == nil {
		config.Logger = &discardLogger{}
	}

	loggerOption := func(c *zk.Conn) {
		c.SetLogger(config.Logger)
	}

	conn, _, err := zk.Connect([]string{config.Addr}, time.Second, loggerOption)
	if err != nil {
		return nil, fmt.Errorf("zookeeper connection failed: %s", err)
	}

	// add Auth if provided
	if config.Auth != nil {
		if err := conn.AddAuth(config.Auth.Scheme, []byte(config.Auth.Auth)); err != nil {
			return nil, fmt.Errorf("zookeeper rejected authentication: %s", err)
		}
	}

	c := &Client{
		conn:   conn,
		acl:    acl,
		config: config,
	}

	// create the base path if provided
	if config.BasePath != "" {
		if err = c.CreateAll(config.BasePath, nil, acl); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, nil
}

// Create the znode.
func (c *Client) Create(path string, value []byte, acls []ACL) error {
	if acls == nil {
		acls = c.DefaultACL(PermAll)
	}
	_, err := c.conn.Create(path, value, int32(0), convertACL(acls))

	return err
}

// CreateEphemeral creates an ephemeral znode.
func (c *Client) CreateEphemeral(path string, value []byte, acls []ACL) error {
	if acls == nil {
		acls = c.DefaultACL(PermAll)
	}
	_, err := c.conn.Create(path, value, zk.FlagEphemeral, convertACL(acls))

	return err
}

// CreateSequential creates a sequential znode.
func (c *Client) CreateSequential(path string, value []byte, acls []ACL) (string, error) {
	if acls == nil {
		acls = c.DefaultACL(PermAll)
	}
	return c.conn.Create(path, value, zk.FlagSequence, convertACL(acls))
}

// CreateEphemeralSequential creates an ephemeral, sequential znode
func (c *Client) CreateEphemeralSequential(path string, value []byte, acls []ACL) (string, error) {
	if acls == nil {
		acls = c.DefaultACL(PermAll)
	}
	return c.conn.CreateProtectedEphemeralSequential(path, value, convertACL(acls))
}

// CreateAll znodes for the path, including parents if necessary.
// value is only put into the last znode child.
func (c *Client) CreateAll(path string, value []byte, acls []ACL) error {
	nodes := strings.Split(path, "/")
	fullPath := ""
	for index, node := range nodes {
		if strings.TrimSpace(node) != "" {
			fullPath += "/" + node
			isLastNode := index+1 == len(nodes)

			// set parent nodes to nil, leaf to value
			// this block reduces round trips by being smart on the leaf create/set
			if exists, _, _ := c.conn.Exists(fullPath); !isLastNode && !exists {
				if err := c.Create(fullPath, nil, acls); err != nil {
					return err
				}
			} else if isLastNode && !exists {
				if err := c.Create(fullPath, value, acls); err != nil {
					return err
				}
			} else if isLastNode && exists {
				if _, err := c.conn.Set(fullPath, value, int32(-1)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Children returns the child znodes for this path.
func (c *Client) Children(path string) ([]string, error) {
	path = c.nodePath(path)
	children, _, err := c.conn.Children(path)
	return children, err
}

// Exists returns whether a znode exists.
func (c *Client) Exists(path string) (bool, error) {
	path = c.nodePath(path)
	exists, _, err := c.conn.Exists(path)
	return exists, err
}

// Get returns the data in the znode.
func (c *Client) Get(path string) ([]byte, error) {
	path = c.nodePath(path)
	body, _, err := c.conn.Get(path)
	return body, err
}

// Put places the data in the znode.
func (c *Client) Put(path string, value []byte) error {
	path = c.nodePath(path)
	_, err := c.conn.Set(path, value, -1)
	return err
}

// Delete removes the znode.
func (c *Client) Delete(path string) error {
	path = c.nodePath(path)
	return c.conn.Delete(path, -1)
}

// DefaultACL returns a slice with a single ACL of the given perms and default ID
// DefaultACL produces an ACL list containing a single ACL which uses the
// provided permissions and the default SchemeID passed through client config.
func (c *Client) DefaultACL(perms int32) []ACL {
	return []ACL{
		{
			Perms:  perms,
			Scheme: c.config.DefaultID.Scheme,
			ID:     c.config.DefaultID.ID,
		},
	}
}

// AuthACL produces an ACL list containing a single ACL which uses the
// provided permissions, with the scheme "auth", and ID "", which is used
// by ZooKeeper to represent any authenticated user.
func AuthACL(perms int32) []ACL {
	return []ACL{
		{
			Perms:  perms,
			Scheme: "auth",
			ID:     "",
		},
	}
}

// WorldACL produces an ACL list containing a single ACL which uses the
// provided permissions, with the scheme "world", and ID "anyone", which
// is used by ZooKeeper to represent any user at all.
func WorldACL(perms int32) []ACL {
	return []ACL{
		{
			Perms:  perms,
			Scheme: "world",
			ID:     "anyone",
		},
	}
}

// DigestACL produces an ACL list containing a single ACL which uses the
// provided permissions, with the scheme "digest" and a digest generated
// for a given user / password.
func DigestACL(perms int32, user, password string) []ACL {
	userPass := []byte(fmt.Sprintf("%s:%s", user, password))
	h := sha1.New()
	if n, err := h.Write(userPass); err != nil || n != len(userPass) {
		panic("SHA1 failed")
	}
	digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return []ACL{
		{
			Perms:  perms,
			Scheme: "digest",
			ID:     user + ":" + digest,
		},
	}
}

// Close terminates the zk Client connection.
func (c *Client) Close() error {
	c.conn.Close()
	return nil
}

// nodePath returns a filepath based on the given path.
func (c *Client) nodePath(p string) string {
	return filepath.Join(c.config.BasePath, p)
}

func convertACL(acls []ACL) []zk.ACL {
	var zkAcls []zk.ACL
	for _, acl := range acls {
		zkAcls = append(zkAcls, zk.ACL(acl))
	}
	return zkAcls
}
