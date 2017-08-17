package cluster

import "testing"

func TestNewCluster(t *testing.T) {
	_, err := NewCluster()
	if err != nil {
		t.Error("Expected no errors getting new cluster, got", err)
	}

}
