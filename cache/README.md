# dcos-go/cache
A simple, local, and goroutine-safe in-memory key-value store.

## Overview
dcos-go/cache is a simple, local, in-memory key-value store for caching objects
for short periods of time. The motivation for this project was the HTTP producer
in [dcos-metrics][dcos-metrics-github] where we have a need to cache a
"snapshot" of a given agent's metrics until the next polling interval.

## Usage

```golang
import "github.com/dcos/dcos-go/cache"

// Basic usage
c := cache.SimpleCache()
c.Set("foo", "fooval")
c.Set("bar", "barval")

c.Get("foo") // fooval
c.Objects()  // map[foo:{fooval} bar:{barval} baz:{bazval}]
c.Size()     // 1
c.Delete("foo")
c.Purge()

// Advanced usage
newMap := make(map[string]interface{})
newMap["foo2"] = "fooval2"
newMap["bar2"] = "barval2"

// Replace (supplant) all objects in the cache with thsoe in newMap
c.Supplant(newMap) // map[foo2:{fooval2} bar2:{barval2}]
```

[dcos-metrics-github]: https://github.com/dcos/dcos-metrics
