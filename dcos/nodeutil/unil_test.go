package nodeutil

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dcos/dcos-go/dcos"
)

func TestDetectIP(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionDetectIP("fixture/detect_ip_good.sh"))
	if err != nil {
		t.Fatal(err)
	}

	ip, err := d.DetectIP()
	if err != nil {
		t.Fatal(err)
	}

	expectIP := net.ParseIP("10.10.0.1")
	if !ip.Equal(expectIP) {
		t.Fatalf("Expect %s. Got %s", expectIP.String(), ip.String())
	}
}

func TestDetectIPFail(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionDetectIP("fixture/detect_ip_bad.sh"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = d.DetectIP(); err == nil {
		t.Fatal("Detect ip returned invalid IP address, but test did not fail")
	}
}

func TestMasterRole(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionMasterRoleFile("fixture/roles/master"))
	if err != nil {
		t.Fatal(err)
	}

	role, err := d.Role()
	if err != nil {
		t.Fatal(err)
	}

	if role != dcos.RoleMaster {
		t.Fatalf("Expect %s. Got %s", dcos.RoleMaster, role)
	}
}

func TestAgentRole(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionAgentRoleFile("fixture/roles/agent"))
	if err != nil {
		t.Fatal(err)
	}

	role, err := d.Role()
	if err != nil {
		t.Fatal(err)
	}

	if role != dcos.RoleAgent {
		t.Fatalf("Expect %s. Got %s", dcos.RoleAgent, role)
	}
}

func TestAgentPublicRole(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionAgentPublicRoleFile("fixture/roles/agent_public"))
	if err != nil {
		t.Fatal(err)
	}

	role, err := d.Role()
	if err != nil {
		t.Fatal(err)
	}

	if role != dcos.RoleAgentPublic {
		t.Fatalf("Expect %s. Got %s", dcos.RoleAgentPublic, role)
	}
}

func TestRoleFail(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionAgentRoleFile("fixture/roles/agent"), OptionAgentPublicRoleFile("fixture/roles/agent_public"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := d.Role(); err == nil {
		t.Fatal("Expect error, got nil")
	}

	d, err = NewNodeInfo(&http.Client{}, OptionMasterRoleFile("fixture/roles/master"), OptionAgentPublicRoleFile("fixture/roles/agent_public"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMesosID(t *testing.T) {
	response := `
	{
	  "id": "abc-def",
	  "slaves": [
	    {
	      "hostname": "10.10.0.1",
	      "id": "ghi-jkl"
	    }
	  ]
	}
	`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	d, err := NewNodeInfo(&http.Client{}, OptionMesosStateURL(ts.URL), OptionMasterRoleFile("fixture/roles/master"),
		OptionDetectIP("fixture/detect_ip_good.sh"))
	if err != nil {
		t.Fatal(err)
	}

	masterID, err := d.MesosID(nil)
	if err != nil {
		t.Fatal(err)
	}

	if masterID != "abc-def" {
		t.Fatalf("Expect master mesos ID: abc-def. Got %s", masterID)
	}

	// Test agent response
	d, err = NewNodeInfo(&http.Client{}, OptionMesosStateURL(ts.URL),
		OptionAgentRoleFile("fixture/roles/agent"), OptionDetectIP("fixture/detect_ip_good.sh"))
	if err != nil {
		t.Fatal(err)
	}

	agentID, err := d.MesosID(nil)
	if err != nil {
		t.Fatal(err)
	}

	if agentID != "ghi-jkl" {
		t.Fatalf("Expect master mesos ID: abc-def. Got %s", agentID)
	}
}

func TestMesosIDFail(t *testing.T) {
	response := "{}"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	d, err := NewNodeInfo(&http.Client{}, OptionMesosStateURL(ts.URL), OptionMasterRoleFile("fixture/roles/master"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := d.MesosID(nil); err == nil {
		t.Fatal("Expect error got nil")
	}
}

func TestIsLeader(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionMasterRoleFile("fixture/roles/master"), OptionLeaderDNSRecord("dcos.io"),
		OptionDetectIP("fixture/detect_ip_good.sh"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.IsLeader()
	if _, ok := err.(ErrNodeInfo); ok == false {
		t.Fatalf("Expect error of type ErrNodeUtil. Got %s", err)
	}
}

func TestClusterID(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionMasterRoleFile("fixture/roles/master"),
		OptionClusterIDFile("fixture/uuid/cluster-id.good"))
	if err != nil {
		t.Fatal(err)
	}

	clusterID, err := d.ClusterID()
	if err != nil {
		t.Fatal(err)
	}

	if clusterID != "b80517ef-4720-43ce-84b3-772066aacf23" {
		t.Fatalf("Expect cluster id b80517ef-4720-43ce-84b3-772066aacf23. Got %s", clusterID)
	}
}

func TestClusterIDInvalidUUID(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionMasterRoleFile("fixture/roles/master"),
		OptionClusterIDFile("fixture/uuid/cluster-id.bad"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.ClusterID()
	if _, ok := err.(ErrNodeInfo); !ok {
		t.Fatalf("Expect error of type ErrNodeInfo. Got %s", err)
	}
}

func TestClusterIDInvalidRole(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, OptionAgentRoleFile("fixture/roles/agent"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = d.ClusterID(); err == nil {
		if _, ok := err.(ErrNodeInfo); !ok {
			t.Fatalf("Expect error of type ErrNodeInfo. Got %s", err)
		}
	}
}
