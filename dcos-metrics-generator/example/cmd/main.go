package main

import (
	"io"
	"time"

	"github.com/uber-go/tally"
	"github.com/dcos/dcos-go/dcos-metrics-generator/statsd"
)

func NewDCOSComponentScope(component string, interval time.Duration) (tally.Scope, io.Closer) {
	tags := map[string]string{
		"region": "aws/us-west-1",
		"zone":   "aws/us-west-1b",
		"ip":     "127.0.0.1",
	}

	reporter := statsd.NewStatsdReporter(
		"127.0.0.1:8125",
		statsd.TagStyle(statsd.TAG_FORMAT_DATADOG))

	options := tally.ScopeOptions{
		Prefix:   "dcos",
		Tags:     tags,
		Reporter: reporter,
	}

	rootScope, closer := tally.NewRootScope(options, interval)

	subScope := rootScope.SubScope(component)

	return subScope, closer
}

func main() {
	scope, closer := NewDCOSComponentScope("edgelb", time.Second)
	defer closer.Close()

	bighand := time.NewTicker(time.Millisecond * 2300)
	littlehand := time.NewTicker(time.Millisecond * 10)
	hugehand := time.NewTicker(time.Millisecond * 5100)

	measureThing := scope.Gauge("thing")
	timings := scope.Timer("timings")
	tickCounter := scope.Counter("ticks")

	// Spin forever, watch report get called
	go func() {
		for {
			select {
			case <-bighand.C:
				measureThing.Update(42.1)
			case <-littlehand.C:
				tickCounter.Inc(1)
			case <-hugehand.C:
				timings.Record(3200 * time.Millisecond)
			}
		}
	}()

	select {}
}
