package nodeutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	// ErrMissingParams is the error raise when the discovery implementations are missing some required parameters.
	ErrMissingParams = errors.New("unable to initialize function, missing required parameters")
)

// NodeDiscoverer is an interface for various agent discovery methods.
type NodeDiscoverer interface {
	Discover() ([]net.IP, error)
}

// DiscoverMastersInExhibitor is a master discovery method from exhibitor.
type DiscoverMastersInExhibitor struct {
	URL     string
	Timeout time.Duration // this must be at least 11 seconds

	// GetFn takes url and timeout and returns a read body, HTTP status code and error.
	GetFn func(string, time.Duration) ([]byte, int, error)

	// Next is the next available discovery method.
	Next NodeDiscoverer
}

// discoverMesosMasters is a working function that does all the work.
func (dm *DiscoverMastersInExhibitor) discoverMesosMasters() ([]net.IP, error) {
	if dm.GetFn == nil {
		return nil, ErrMissingParams
	}

	// must be at least 11 seconds because of exhibitor bug.
	if dm.Timeout < time.Second*11 {
		dm.Timeout = time.Duration(time.Second * 11)
	}

	body, statusCode, err := dm.GetFn(dm.URL, dm.Timeout)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s failed, status code: %d", dm.URL, statusCode)
	}

	// is a exhibitor response format
	type exhibitorNodeResponse struct {
		Code        int
		Description string
		Hostname    string
		IsLeader    bool
	}

	var exhibitorNodesResponse []exhibitorNodeResponse
	if err := json.Unmarshal([]byte(body), &exhibitorNodesResponse); err != nil {
		return nil, err
	}

	if len(exhibitorNodesResponse) == 0 {
		return nil, errors.New("master nodes not found in exhibitor")
	}

	nodes := []net.IP{}
	for _, exhibitorNodeResponse := range exhibitorNodesResponse {
		ip := net.ParseIP(exhibitorNodeResponse.Hostname)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip address returned by exhibitor %s", exhibitorNodeResponse.Hostname)
		}
		nodes = append(nodes, ip)
	}
	return nodes, nil
}

// Discover will try to get the master nodes from exhibitor. It will return nodes on success, otherwise it'll try to
// find a next available discovery method and execute it.
func (dm *DiscoverMastersInExhibitor) Discover() ([]net.IP, error) {
	nodes, err := dm.discoverMesosMasters()
	if err == nil && len(nodes) > 0 {
		return nodes, nil
	}

	if dm.Next != nil {
		return dm.Next.Discover()
	}

	return nil, err
}
