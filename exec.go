package docker

import (
	"io"

	"github.com/docker/docker/pkg/stdcopy"
	dockType "github.com/docker/engine-api/types"
	"github.com/jcelliott/lumber"
	"golang.org/x/net/context"
)

func ExecStart(containerID string, cmd []string, stdIn, stdOut, stdErr bool) (dockType.ContainerExecCreateResponse, dockType.HijackedResponse, error) {
	config := dockType.ExecConfig{
		Tty:          true,
		Cmd:          cmd,
		Container:    containerID,
		AttachStdin:  stdIn,
		AttachStdout: stdOut,
		AttachStderr: stdErr,
	}
	exec, err := client.ContainerExecCreate(context.Background(), config)
	if err != nil {
		return exec, dockType.HijackedResponse{}, err
	}
	resp, err := client.ContainerExecAttach(context.Background(), exec.ID, config)
	return exec, resp, err
}

func ExecInspect(id string) (dockType.ContainerExecInspect, error) {
	return client.ContainerExecInspect(context.Background(), id)
}

func ExecPipe(resp dockType.HijackedResponse, inStream io.Reader, outStream, errorStream io.Writer) error {
	var err error
	receiveStdout := make(chan error, 1)
	if outStream != nil || errorStream != nil {
		go func() {
			if outStream != nil {
				_, err = io.Copy(outStream, resp.Reader)
			} else {
				// use a multicopy protocol made by docker
				_, err = stdcopy.StdCopy(outStream, errorStream, resp.Reader)
			}
			lumber.Debug("[hijack] End of stdout")
			receiveStdout <- err
		}()
	}

	stdinDone := make(chan struct{})
	go func() {
		if inStream != nil {
			io.Copy(resp.Conn, inStream)
			lumber.Debug("[hijack] End of stdin")
		}

		if err := resp.CloseWrite(); err != nil {
			lumber.Debug("Couldn't send EOF: %s", err)
		}
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		if err != nil {
			lumber.Debug("Error receiveStdout: %s", err)
			return err
		}
	case <-stdinDone:
		if outStream != nil || errorStream != nil {
			if err := <-receiveStdout; err != nil {
				lumber.Debug("Error receiveStdout: %s", err)
				return err
			}
		}
	}

	return nil
}

// resize the exec.
func ContainerExecResize(id string, height, width int) error {
	return client.ContainerExecResize(context.Background(), dockType.ResizeOptions{id, height, width})
}
