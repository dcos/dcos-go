package api

import "github.com/dcos/dcos-go/dcos-log/router"

func loadRoutes() []router.Route {
	return []router.Route{
		// wait for the new logs, server will not close the connection
		{
			URL:     "/stream",
			Handler: streamingServerSSEHandler,
			Headers: []string{"Accept", "text/event-stream"},
		},
		{
			URL:     "/stream",
			Handler: streamingServerJSONHandler,
			Headers: []string{"Accept", "application/json"},
		},
		{
			URL:     "/stream",
			Handler: streamingServerTextHandler,
			Headers: []string{"Accept", "text/(plain|html)"},
		},

		// get a range of logs, do not wait
		{
			URL:     "/logs",
			Handler: rangeServerSSEHandler,
			Headers: []string{"Accept", "text/event-stream"},
		},
		{
			URL:     "/logs",
			Handler: rangeServerJSONHandler,
			Headers: []string{"Accept", "application/json"},
		},
		{
			URL:     "/logs",
			Handler: rangeServerTextHandler,
			Headers: []string{"Accept", "text/(plain|html)"},
		},

		// TODO(mnaboka): indexHandler is not supposed to be here. Remove it.
		// read index.html off the filesystem
		{
			URL:     "/",
			Handler: indexHandler,
			Headers: []string{"Accept", "text/(plain|html)"},
		},
	}
}
