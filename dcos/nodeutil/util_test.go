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
	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionDetectIP("fixture/detect_ip_good.sh"))
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
	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionDetectIP("fixture/detect_ip_bad.sh"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = d.DetectIP(); err == nil {
		t.Fatal("Detect ip returned invalid IP address, but test did not fail")
	}
}

func TestMesosID(t *testing.T) {
	response := `
	{
	  "id": "abc-def",
	  "slaves": [
	    {
	      "pid": "slave(1)@10.10.0.1:5051",
	      "id": "ghi-jkl"
	    }
	  ]
	}
	`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionMesosStateURL(ts.URL),
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
	d, err = NewNodeInfo(&http.Client{}, dcos.RoleAgent, OptionMesosStateURL(ts.URL),
		OptionDetectIP("fixture/detect_ip_good.sh"))
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

	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionMesosStateURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := d.MesosID(nil); err == nil {
		t.Fatal("Expect error got nil")
	}
}

func TestIsLeader(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionLeaderDNSRecord("dcos.io"),
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
	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionClusterIDFile("fixture/uuid/cluster-id.good"))
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
	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionClusterIDFile("fixture/uuid/cluster-id.bad"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.ClusterID()
	if _, ok := err.(ErrNodeInfo); !ok {
		t.Fatalf("Expect error of type ErrNodeInfo. Got %s", err)
	}
}

func TestClusterIDInvalidRole(t *testing.T) {
	d, err := NewNodeInfo(&http.Client{}, dcos.RoleAgent)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = d.ClusterID(); err == nil {
		if _, ok := err.(ErrNodeInfo); !ok {
			t.Fatalf("Expect error of type ErrNodeInfo. Got %s", err)
		}
	}
}

func TestContextWithHeaders(t *testing.T) {
	header := http.Header{}
	header.Add("TEST", "123")

	ctx := NewContextWithHeaders(nil, header)
	if ctx == nil {
		t.Fatal("Context shouldn't be nil")
	}

	headerFromContext, ok := HeaderFromContext(ctx)
	if !ok {
		t.Fatal("header not found in context")
	}

	if value := headerFromContext.Get("TEST"); value != "123" {
		t.Fatalf("Expect header `TEST:123`. Got %+v", headerFromContext)
	}
}
