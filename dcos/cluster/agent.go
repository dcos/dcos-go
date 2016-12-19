package cluster

// Agent{} defines a DC/OS agent host
type Agent struct {
	State AgentState
	Host
}

/*
cluster := dcos.cluster.New()
for agent := range cluster.AgentList {
	ipAddr := agent.GetIPAddress()
}
*/
