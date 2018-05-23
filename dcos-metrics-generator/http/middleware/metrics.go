package middleware

import (
	"context"
	"net/http"
	"time"
	"github.com/uber-go/tally"
	"io"
)

type statsKey int

const (
	metricsKey = statsKey(iota)
)

// ScopeCloser is an interface which implements io.Closer and tally.Scope
type ScopeCloser interface {
	io.Closer
	tally.Scope
}

func WithScopeContext(ctx context.Context, scope ScopeCloser) context.Context {
	return context.WithValue(ctx, metricsKey, scope)
}

func FromStatsContext(ctx context.Context) (ScopeCloser, bool) {
	instance := ctx.Value(metricsKey)
	stats, ok := instance.(ScopeCloser)
	return stats, ok
}


func StatsMiddleware(next http.Handler, scope tally.Scope) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		rec := &statusRecorder{
			w, 200,
		}

		next.ServeHTTP(rec, r)

		scope = scope.Tagged(map[string]string{
			"endpoint": r.URL.Path,
		})

		scope.Timer("response_time").Record(time.Since(t))
		scope.Gauge("status_code").Update(float64(rec.status))
	})
}
