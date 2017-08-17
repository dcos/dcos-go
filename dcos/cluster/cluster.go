package cluster

import (
	"net"
	"sync"
)

// Cluster{} defines a DC/OS cluster
type Cluster struct {
	Masters []Master
	sync.Mutex
}

// NewCluster() returns a new instance of a cluster type with pre-loaded
// information for hostInfo
func NewCluster() (Cluster, error) {
	cluster := Cluster{
		Masters: []Master{},
	}

	if err := cluster.setMasterIPs(); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func (c *Cluster) setMasterIPs() error {
	// query master.mesos and get list of IPs
	ips, err := net.LookupIP("master.mesos")
	if err != nil {
		return err
	}

	for i, ip := range ips {
		c.Masters[0].Host.IPAddress = ip
	}

	return nil
}

// setHostInfo sets the host information for each host in the DC/OS
// cluster by qurying Mesos state.json and
// agent state from the Mesos state.json endpoint
func (c *Cluster) setHostInfo() error {
	return nil
}
