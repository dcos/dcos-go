package nodeutil

// State stands for mesos state.json available via /mesos/master/state.json
type State struct {
	ID         string      `json:"id"`
	Slaves     []Slave     `json:"slaves"`
	Frameworks []Framework `json:"frameworks"`
}

// Slave is a field in state.json
type Slave struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Pid      string `json:"pid"`
}

// Framework is a field in state.json
type Framework struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	PID   string `json:"pid"`
	Role  string `json:"role"`
	Tasks []Task `json:"tasks"`
}

// Task is a field in state.json
type Task struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	FrameworkID string `json:"framework_id"`
	ExecutorID  string `json:"executor_id"`
	SlaveID     string `json:"slave_id"`
	State       string `json:"state"`
	Role        string `json:"role"`

	Statuses []struct {
	} `json:"statuses"`
}
