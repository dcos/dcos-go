package cluster

import "net"

/* Host expresses a physcial node or virtual machine. Host should
return data which is specific to a host resource. Data which comes
from services which may be running on a host are out of scope for
a host interface.
*/
type Host struct {
	IPAddress net.IP
	Hostname  string
	MesosID   string
	MesosRole string
}

func (h *Host) getIPAddress(ipDetectPath string) error { return nil }
func (h *Host) getHostname() error                     { return nil }
func (h *Host) getMesosID() error                      { return nil }
func (h *Host) getMesosRole() error                    { return nil }
