package exec

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ce, err := Run("ls", []string{"-la"}, Timeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	buffer := new(bytes.Buffer)
	io.Copy(buffer, ce)
	err = <-ce.Done
	if err != nil {
		t.Fatalf("Return should be nil. Got %s", err)
	}

	debugOutput := buffer.String()
	scanner := bufio.NewScanner(buffer)
	var foundString int
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "exec.go") || strings.Contains(scanner.Text(), "exec_test.go") {
			foundString++
		}
	}

	if foundString != 2 {
		t.Fatalf("Expecting `exec.go` and `exec_test.go` in output. Got: %s", debugOutput)
	}
}

func TestRunTimeout(t *testing.T) {
	ce, err := Run("bash", []string{"./fixture/infinite.sh"}, Timeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	buffer := new(bytes.Buffer)
	io.Copy(buffer, ce)
	err = <-ce.Done
	if err != context.DeadlineExceeded {
		t.Fatalf("Return should be %s. Got %s", context.DeadlineExceeded, err)
	}
}

func TestRunCancel(t *testing.T) {
	ce, err := Run("bash", []string{"./fixture/infinite.sh"}, Timeout(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(time.Second)
		ce.Cancel()
	}()
	buffer := new(bytes.Buffer)
	io.Copy(buffer, ce)
	err = <-ce.Done
	if err != context.Canceled {
		t.Fatalf("Expected %s .Got %s", context.Canceled, err)
	}
}

func TestBadReturnCode(t *testing.T) {
	ce, err := Run("command_no_found", []string{"abc"}, Timeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	buffer := new(bytes.Buffer)
	io.Copy(buffer, ce)
	err = <-ce.Done
	if !strings.Contains(err.Error(), "command_no_found") {
		t.Fatalf("Expected `command_no_found` in error output.Got: %s", err)
	}
}
