package nodeutil

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"
)

func isInList(ip net.IP, ips []net.IP) bool {
	for _, i := range ips {
		if i.Equal(ip) {
			return true
		}
	}
	return false
}

func TestDiscoverMastersInExhibitor_Discover(t *testing.T) {
	nodes := []struct {
		Hostname string
	}{
		{
			Hostname: "127.0.0.1",
		},
		{
			Hostname: "127.0.0.2",
		},
		{
			Hostname: "127.0.0.3",
		},
	}

	marshaledNodes, err := json.Marshal(nodes)
	if err != nil {
		t.Fatalf("Error marshaling nodes: %s", err)
	}

	mastersDiscovery := DiscoverMastersInExhibitor{
		URL: "http://127.0.0.1/exhibitor",
		GetFn: func(url string, timeout time.Duration) ([]byte, int, error) {
			if url != "http://127.0.0.1/exhibitor" {
				t.Fatalf("Invalid url %s", url)
			}

			return marshaledNodes, 200, nil
		},
	}

	ips, err := mastersDiscovery.Discover()
	if err != nil {
		t.Fatalf("Expecting master nodes, got an error: %s", err)
	}

	for _, expectedIP := range []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("127.0.0.2"),
		net.ParseIP("127.0.0.3"),
	} {
		if !isInList(expectedIP, ips) {
			t.Fatal("OOPS")
		}
	}

	// test bad case
	mastersDiscovery = DiscoverMastersInExhibitor{
		URL: "http://127.0.0.1/exhibitor",
		GetFn: func(url string, timeout time.Duration) ([]byte, int, error) {
			if url != "http://127.0.0.1/exhibitor" {
				t.Fatalf("Invalid url %s", url)
			}

			return nil, 500, fmt.Errorf("Some error")
		},
	}

	ips, err = mastersDiscovery.Discover()
	if err == nil {
		t.Fatal("Expecing an error, got nil")
	}

	if ips != nil {
		t.Fatalf("Expecting ips to be nil, got %+v", nodes)
	}
}
