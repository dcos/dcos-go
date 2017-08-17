package cluster

// Master{} defines a DC/OS master host
type Master struct {
	State MasterState
	Host
}
