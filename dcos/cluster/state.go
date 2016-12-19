package cluster

type AgentState struct {
	Version   string `json:"version"`
	GitSHA    string `json:"git_sha"`
	GitBranch string `json:"git_branch"`
	GitTag    string `json:"git_tag"`
	BuildDate string `json:"build_date"`
	BuildTime int    `json:"build_time"`
	BuildUser string `json:"build_user"`
	StartTime int    `json:"start_time"`
	ID        string `json:"id"`
	PID       string `json:"pid"`
	Hostname  string `json:"hostname"`
	Resources struct {
		Ports string `json:"ports"`
		Mem   int64  `json:"mem"`
		Disk  int64  `json:"disk"`
		CPUS  int    `json:"cpus"`
	} `json:"resources"`
	Attributes          interface{} `json:"attributes"`
	MasterHostname      string      `json:"master_hostname"`
	LogDir              string      `json:"log_dir"`
	ExternalLogFile     string      `json:"external_log_file"`
	Frameworks          []string    `json:"frameworks"`
	CompletedFrameworks []string    `json:"completed_frameworks"`
	Flags               struct {
		FrameworkSorter string `json:"framework_sorter"`
		// TODO (malnick) add the rest of the flags...
	} `json:"flags"`
}

type MasterState struct {
	Version           string `json:"version"`
	GitSHA            string `json:"git_sha"`
	GitBranch         string `json:"git_branch"`
	GitTag            string `json:"git_tag"`
	BuildDate         string `json:"build_date"`
	BuildTime         int    `json:"build_time"`
	BuildUser         string `json:"build_user"`
	StartTime         int    `json:"start_time"`
	ElectedTime       int    `json:"elected_time"`
	ID                string `json:"id"`
	PID               string `json:"pid"`
	Hostname          string `json:"hostname"`
	ActivatedSlaves   int    `json:"activated_slaves"`
	DeactivatedSlaves int    `json:"deactivated_slaves"`
	Cluster           string `json:"cluster"`
	Leader            string `json:"leader"`
	LogDir            string `json:"log_dir"`
	ExternalLogFile   string `json:"external_log_file"`
	Flags             struct {
		FrameworkSorter string `json:"framework_sorter"`
		// TODO (malnick) add the rest of the flags...
	} `json:"flags"`
	Slaves                 []struct{} `json:"slaves"`
	Frameworks             []struct{} `json:"frameworks"`
	CompletedFrameworks    []struct{} `json:"completed_frameworks"`
	OrphanTasks            []struct{} `json:"orphan_tasks"`
	UnregisteredFrameworks []struct{} `json:"unregistered_frameworks"`
}
