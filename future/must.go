package future

// HandleMust may be overridden by downstream consumers w/ custom handlers for
// assertion failure.
var HandleMust = func(err error) { panic(err) }

func Must(v interface{}, err error) interface{} {
	if err != nil {
		HandleMust(err)
	}
	return v
}
