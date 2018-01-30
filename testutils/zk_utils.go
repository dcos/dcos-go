package testutils

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// StartZookeeper starts a new zookeeper container.
func StartZookeeper() (*ZkControl, error) {
	dcli, err := DockerClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not get docker client")
	}
	image := "docker.io/jplock/zookeeper:3.4.10"

	if err := pullDockerImage(dcli, image); err != nil {
		return nil, err
	}

	// the container IP is not routable on Darwin, thus needs port
	// mapping for the container.
	hostConfig := &container.HostConfig{}
	if runtime.GOOS == "darwin" {
		hostConfig.PortBindings = nat.PortMap{
			"2181/tcp": []nat.PortBinding{{
				HostIP:   "0.0.0.0",
				HostPort: "2181",
			}},
		}
	}

	r, err := dcli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:      image,
			Entrypoint: []string{"/opt/zookeeper/bin/zkServer.sh"},
			Cmd:        []string{"start-foreground"},
		},
		hostConfig,
		nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not create zk container")
	}

	// create a teardown that will be used here to try to tear down the
	// container if anything fails in setup
	cleanup := func() {
		removeContainer(dcli, r.ID)
	}

	// start the container
	if err := dcli.ContainerStart(context.Background(), r.ID, types.ContainerStartOptions{}); err != nil {
		cleanup()
		return nil, errors.Wrap(err, "could not start zk container")
	}

	info, err := dcli.ContainerInspect(context.Background(), r.ID)
	if err != nil {
		cleanup()
		return nil, errors.Wrap(err, "could not inspect container")
	}

	var addr string
	if runtime.GOOS == "darwin" {
		addr = "127.0.0.1:2181"
	} else {
		addr = info.NetworkSettings.IPAddress + ":2181"
	}

	done := make(chan struct{})
	defer close(done)

	connected := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					time.Sleep(1)
					continue
				}
				fmt.Println("successfully connected to ZK at", addr)
				conn.Close()
				close(connected)
				return
			}

		}
	}()
	timeout := 10 * time.Second
	select {
	case <-connected:
	case <-time.After(timeout):
		cleanup()
		return nil, errors.Errorf("could not connect to zookeeper in %s", timeout)
	}
	control := &ZkControl{
		dockerClient: dcli,
		containerID:  r.ID,
		addr:         addr,
	}
	return control, nil
}
