package cluster

import (
	"sync"
	"time"
)

type Option func(*clusterConfig) error

type clusterConfig struct {
	sync.Mutex
	cache           bool
	ipDetectPath    string
	ipDetectTimeout time.Duration
}
