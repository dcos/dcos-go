package nodeutil

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"context"
	"github.com/dcos/dcos-go/dcos"
	"io/ioutil"
	"strings"
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

func TestMesosRuntimeShortCanonicalID(t *testing.T) {
	expectedAgentID := "db10f9b1-5b82-4187-aa47-4fbcefc7cdca-S1"
	expectedFrameworkID := "db10f9b1-5b82-4187-aa47-4fbcefc7cdca-0000"
	expectedExecutorID := "single-mesos-container.c1f5ae3f-b81f-11e7-a9ac-52ad791ffaa8"
	expectedContainerID := "1a69d257-48ca-4d3b-aead-332ad881fcc7"

	if err := testCanonicalID("single-mesos-container", expectedAgentID, expectedFrameworkID, expectedExecutorID, expectedContainerID); err != nil {
		t.Fatal(err)
	}
}

func TestMesosRuntimeLongCanonicalID(t *testing.T) {
	expectedAgentID := "db10f9b1-5b82-4187-aa47-4fbcefc7cdca-S1"
	expectedFrameworkID := "db10f9b1-5b82-4187-aa47-4fbcefc7cdca-0000"
	expectedExecutorID := "single-mesos-container.c1f5ae3f-b81f-11e7-a9ac-52ad791ffaa8"
	expectedContainerID := "1a69d257-48ca-4d3b-aead-332ad881fcc7"
	if err := testCanonicalID("single-mesos-container.c1f5ae3f-b81f-11e7-a9ac-52ad791ffaa8", expectedAgentID, expectedFrameworkID, expectedExecutorID, expectedContainerID); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalTaskIDNotFound(t *testing.T) {
	if err := testCanonicalID("foobar", "", "", "", ""); err != ErrTaskNotFound {
		t.Fatalf("error must be %s. Got %s", ErrTaskNotFound, err)
	}
}

func TestPodCanonicalID(t *testing.T) {
	expectedAgentID := "db10f9b1-5b82-4187-aa47-4fbcefc7cdca-S1"
	expectedFrameworkID := "db10f9b1-5b82-4187-aa47-4fbcefc7cdca-0000"
	expectedExecutorID := "instance-parent-pod.da6ef080-b81f-11e7-a9ac-52ad791ffaa8"
	expectedContainerID := "e7ed292a-8390-4da4-8c2a-c13b554e2c2a-1eb53d03-e8f2-4de7-8a51-be17b42a3a29"
	if err := testCanonicalID("container-1", expectedAgentID, expectedFrameworkID, expectedExecutorID, expectedContainerID); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalIDSameNameTasks(t *testing.T) {
	err := testCanonicalID("test123", "", "", "", "")
	if err == nil {
		t.Fatal("expecting error. Got nil")
	}

	expectedError := "found more then 1 task with name test123: [test123-123 test123-345]"
	if err.Error() != expectedError {
		t.Fatalf("expect error %s. Got %s", expectedError, err)
	}
}

func testCanonicalID(task, expectedAgentID, expectedFrameworkID, expectedExecutorID, expectedContainerID string) error {
	state, err := ioutil.ReadFile("fixture/state.json")
	if err != nil {
		return err
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(state))
	}))
	defer ts.Close()

	d, err := NewNodeInfo(&http.Client{}, dcos.RoleMaster, OptionMesosStateURL(ts.URL))
	if err != nil {
		return err
	}

	cID, err := d.TaskCanonicalID(context.TODO(), task)
	if err != nil {
		return err
	}

	if cID.AgentID != expectedAgentID {
		return fmt.Errorf("expecting agent id %s. Got %s", expectedAgentID, cID.AgentID)
	}

	if cID.FrameworkID != expectedFrameworkID {
		return fmt.Errorf("expecting framework id %s. Got %s", expectedFrameworkID, cID.FrameworkID)
	}

	if cID.ExecutorID != expectedExecutorID {
		return fmt.Errorf("expecting executor id %s. Got %s", expectedExecutorID, cID.ExecutorID)
	}

	if strings.Join(cID.ContainerIDs, "-") != expectedContainerID {
		return fmt.Errorf("expecting task id: %s. Got %s", expectedContainerID, cID.ContainerIDs)
	}

	return nil
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
