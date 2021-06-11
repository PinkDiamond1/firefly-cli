package docker

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func RunDockerCommand(workingDir string, showCommand bool, pipeStdout bool, command ...string) error {
	dockerCmd := exec.Command("docker", command...)
	dockerCmd.Dir = workingDir
	return runCommand(dockerCmd, showCommand, pipeStdout, command...)
}

func RunDockerComposeCommand(workingDir string, showCommand bool, pipeStdout bool, command ...string) error {
	dockerCmd := exec.Command("docker-compose", command...)
	dockerCmd.Dir = workingDir
	return runCommand(dockerCmd, showCommand, pipeStdout, command...)
}

func runCommand(cmd *exec.Cmd, showCommand bool, pipeStdout bool, command ...string) error {
	if showCommand {
		fmt.Println(cmd.String())
	}
	outputBuff := strings.Builder{}
	stdoutChan := make(chan string)
	stderrChan := make(chan string)
	errChan := make(chan error)
	go pipeCommand(cmd, stdoutChan, stderrChan, errChan)

outputCapture:
	for {
		select {
		case s, ok := <-stdoutChan:
			if pipeStdout {
				if !ok {
					break outputCapture
				}
				fmt.Print(s)
			} else {
				outputBuff.WriteString(s)
			}
		case s, ok := <-stderrChan:
			if !ok {
				break outputCapture
			}
			if pipeStdout {
				fmt.Print(s)
			} else {
				outputBuff.WriteString(s)
			}
		case err := <-errChan:
			return err
		}
	}
	cmd.Wait()
	statusCode := cmd.ProcessState.ExitCode()
	if statusCode != 0 {
		return fmt.Errorf("%s\nFailed [%d] %s", strings.Join(cmd.Args, " "), statusCode, outputBuff.String())
	}
	return nil
}

func pipeCommand(cmd *exec.Cmd, stdoutChan chan string, stderrChan chan string, errChan chan error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errChan <- err
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		errChan <- err
		return
	}
	cmd.Start()
	go readPipe(stdout, stdoutChan, errChan)
	go readPipe(stderr, stderrChan, errChan)
}

func readPipe(pipe io.ReadCloser, outputChan chan string, errChan chan error) {
	buf := bufio.NewReader(pipe)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				close(outputChan)
				return
			} else {
				errChan <- err
				close(outputChan)
				return
			}
		} else {
			outputChan <- line
		}
	}
}