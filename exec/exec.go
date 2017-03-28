package exec

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

// dcos-go/exec is a os/exec wrapper. It implements io.Reader and can be used to read both STDOUT and STDERR.
//
// Usage:
// ce := exec.Run("bash", []string{"infinite.sh"}, exec.Timeout(3 * time.Second))
//
// io.Copy(os.Stdout, ce)
// err := <- ce.Done
// if err != nil {
// 	log.Error(err)
// }

// ErrInvalidTimeout is the error returned by `func Timeout` if a non-positive timeout option specified.
var ErrInvalidTimeout = errors.New("Timeout cannot be negative or empty")

// CommandExecutor is a structure returned by exec.Run
// Cancel can be used by a user to interrupt a command execution.
// Done is a channel the user can read in order to retrieve execution status. Possible statuses:
//  <nil> command executed successfully, returned 0 exit code
//  <exit status N> where N is non 0 exit status.
//  <context deadline exceeded> means timeout was reached and command was killed.
//  <context canceled>  means that command was canceled by a user.
type CommandExecutor struct {
	Done chan error

	done chan error
	pipe *io.PipeReader
}

// Read implements the io.Reader.
// CommandExecutor will read from stdout and stderr
func (c *CommandExecutor) Read(p []byte) (int, error) {
	return c.pipe.Read(p)
}

// Run spawns the given command and returns a handle to the running process in the form
// of a CommandExecutor.
func Run(ctx context.Context, command string, arg []string) (*CommandExecutor, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// by default Cancel is spineless unless someone configures an option to enable it
	commandExecutor := &CommandExecutor{Done: make(chan error, 1), done: make(chan error, 1)}

	cmd := exec.CommandContext(ctx, command, arg...)
	go func() {
		var err error
		defer func() { commandExecutor.Done <- err }()

		select {
		case <-ctx.Done():
			err = ctx.Err()
		case err = <-commandExecutor.done:
		}
	}()

	// Create a new PIPE.
	// stdout and stderr will be both redirected to this pipe. When the command is executed / cancelled or timeout
	// reached the pipe will be closed, unblocking the reader.
	r, w := io.Pipe()
	cmd.Stdout = w
	cmd.Stderr = w
	commandExecutor.pipe = r

	// execute the command in the goroutine.
	go func() {
		defer w.Close()
		commandExecutor.done <- cmd.Run()
	}()

	return commandExecutor, nil
}
