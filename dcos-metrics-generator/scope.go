package dcos_metrics_generator

import (
	"time"
	"io"

	"github.com/uber-go/tally"
	"github.com/dcos/dcos-go/dcos-metrics-generator/statsd"
	"context"
	"runtime"
	"strconv"
)

const (
	scopePrefix = "dcos"
)

func NewNotifyCloser(ctx context.Context, cancel context.CancelFunc, closer io.Closer) io.Closer {
	return &notifyCloser{
		parent: closer,
		ctx: ctx,
		cancel: cancel,
	}
}

type notifyCloser struct {
	parent io.Closer
	ctx context.Context
	cancel context.CancelFunc
}

func (n notifyCloser) Close() error {
	n.cancel()
	return n.parent.Close()
}

// NewDCOSComponentScope returns a new instance of DC/OS component scope and a closer
// takes the following arguments:
// component - name used in metrics
// interval - flushing interval
// tags - arbitrary tags to be sent with every metric
// reporter - implementation of tally.StatsReporter, if nil is used, default statsd reporter will be used.
func NewDCOSComponentScope(component string, interval time.Duration, tags map[string]string,
						   reporter tally.StatsReporter, reportRuntimeMetrics bool) (tally.Scope, io.Closer) {
	if reporter == nil {
		reporter = statsd.NewStatsdReporter("127.0.0.1:8125", statsd.TagStyle(statsd.TAG_FORMAT_DATADOG))
	}

	options := tally.ScopeOptions{
		Prefix:   scopePrefix,
		Tags:     tags,
		Reporter: reporter,
	}

	rootScope, closer := tally.NewRootScope(options, interval)
	subScope := rootScope.SubScope(component)

	ctx, cancel := context.WithCancel(context.Background())

	notifyCloser := NewNotifyCloser(ctx, cancel, closer)
	if reportRuntimeMetrics {
		go func() {
			for {
				select {
				case <- ctx.Done():
					return
				case <- time.After(interval):
					reportSystemMetrics(subScope)
				}
			}
		}()
	}


	return subScope, notifyCloser
}

func reportSystemMetrics(scope tally.Scope) {
	scope.Gauge("num_goroutines").Update(float64(runtime.NumGoroutine()))

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	scope.Tagged(map[string]string{
		"mem_alloc": strconv.FormatUint(memStats.Alloc, 10),
		"mem_lookups": strconv.FormatUint(memStats.Lookups, 10),
		"mem_mallocs": strconv.FormatUint(memStats.Mallocs, 10),
		"mem_frees": strconv.FormatUint(memStats.Frees, 10),
	}).Gauge("mem_sys").Update(float64(memStats.Sys))
}
