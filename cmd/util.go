package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"bytes"
	log "github.com/Sirupsen/logrus"
	"io"
	"strings"
	"time"
)

type AnsibleConfig struct {

	// HostPath is the path to the directory containing the role
	// on the host machine, which could be anywhere.
	HostPath string

	// RemotePath is the path to the roles folder on the container
	// which should represent the roles folder (ie /etc/ansible/roles)
	RemotePath string

	// The path to the requirements file relative to HostPath.
	// Requirements will not attempt installation if the field
	// does not have a value (when value == "")
	RequirementsFile string

	// The path to the playbook located in the tests file relative to
	// HostPath (ie HostPath/tests/playbook.yml)
	PlaybookFile string
}

// Container is an interface which allows
// a user from plugging in a Distribution
// to use these functions to run Ansible tests.
// Details on
type Container interface {
	run(config *AnsibleConfig)
	install(config *AnsibleConfig)
	kill()
	test(config *AnsibleConfig)
}

// Checks if the specified container is running.
func is_running() bool {
	// Users should not be able to re-run containers with the same name...
	out, err := docker_exec([]string{
		"ps",
		"-f",
		"status=running",
		"--format",
		"'{{.Names}}'",
	}, false)

	if err != nil {
		return false
	}

	if strings.Contains(out, containerID) {
		return true
	}

	return false
}

// docer_exec will execute a command to the docker binary
// and use the input args as arguments for that process.
// You can request output be printed using the bool stdout.
func docker_exec(args []string, stdout bool) (string, error) {

	// Generate the command, based on input.
	cmd := exec.Cmd{}
	cmd.Path = docker
	cmd.Args = []string{docker}

	// Add our arguments to the command.
	for _, arg := range args {
		cmd.Args = append(cmd.Args, arg)
	}

	// If configured, print to os.Stdout.
	if stdout {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
	}

	// Create a buffer for the output.
	var out bytes.Buffer
	multi := io.MultiWriter(&out)

	if stdout {
		multi = io.MultiWriter(&out, os.Stdout)
	}

	// Assign the output to the writer.
	cmd.Stdout = multi

	// Check the errors, return as needed.
	if err := cmd.Run(); err != nil {
		log.Errorln(err)
		return out.String(), err
	}
	cmd.Wait()

	// Return out output as a string.
	return out.String(), nil
}

// run will launch a new container (containerID) using
// the fields in a AnsibleConfig struct.
func (dist *Distribution) run(config *AnsibleConfig) {

	if containerID == "" {
		containerID = fmt.Sprint(time.Now().Unix())
	}

	log.Printf("Running %v", containerID)

	var run_options string
	if dist.Privileged {
		run_options += fmt.Sprintf("--privileged")
	}

	docker_exec([]string{
		"run",
		"--detach",
		fmt.Sprintf("--name=%v", containerID),
		fmt.Sprintf("--volume=%v", dist.Volume),
		fmt.Sprintf("--volume=%v:%v", config.HostPath, config.RemotePath),
		run_options,
		dist.Container,
		dist.Initialise,
	}, false)
}

// Install will install the requirements if the file is configured.
func (dist *Distribution) install(config *AnsibleConfig) {

	if config.RequirementsFile != "" {
		req := fmt.Sprintf("%v/%v", config.RemotePath, config.RequirementsFile)
		log.Printf("Installing requirements from %v\n", req)
		docker_exec([]string{
			"exec",
			"--tty",
			containerID,
			"ansible-galaxy",
			"install",
			fmt.Sprintf("-r %v", req),
		}, true)
	} else {
		log.Warnln("Requirements file is not configured (empty/null), skipping...")
	}
}

// Kill will stop the container and remove it.
func kill() {

	if containerID != "" {

		if is_running() {

			log.Printf("Stopping %v\n", containerID)
			docker_exec([]string{
				"stop",
				containerID,
			}, false)
		} else {
			log.Errorf("container %v is not running\n", containerID)
		}

	} else {
		log.Errorln("container name was not specified")
	}

}

func (dist *Distribution) test_syntax(config *AnsibleConfig) {

	// Ansible syntax check.
	log.Infoln("Checking role syntax...")
	docker_exec([]string{
		"exec",
		"--tty",
		containerID,
		"ansible-playbook",
		fmt.Sprintf("%v/tests/%v", config.RemotePath, config.PlaybookFile),
		"--syntax-check",
	}, true)

	log.Infoln("PASS")
}
func (dist *Distribution) test_role(config *AnsibleConfig) {

	// Test role.
	log.Infoln("Running the role...")
	docker_exec([]string{
		"exec",
		"--tty",
		containerID,
		"ansible-playbook",
		fmt.Sprintf("%v/tests/%v", config.RemotePath, config.PlaybookFile),
	}, true)
}

func (dist *Distribution) test_idempotence(config *AnsibleConfig) {

	// Test role idempotence.
	log.Infoln("Testing role idempotence...")
	out, _ := docker_exec([]string{
		"exec",
		"--tty",
		string(containerID),
		"ansible-playbook",
		fmt.Sprintf("%v/tests/%v", config.RemotePath, config.PlaybookFile),
	}, true)

	idempotence := idempotence_result(out)
	if idempotence {
		log.Infoln("Idempotence test: PASS")
	} else {
		log.Errorln("Idempotence test: FAIL")
	}
}

func idempotence_result(output string) bool {

	lines := strings.Split(output, "\n")

	changed := ""
	failed := ""

	for _, line := range lines {
		if strings.Contains(line, "=") {
			f := strings.Split(line, "=")
			if strings.Contains(line, "changed=") {
				changed = strings.Split(f[2], " ")[0]
			}
			if strings.Contains(line, "failed=") {
				failed = strings.Split(f[4], " ")[0]
			}
		}
	}

	if failed != "0" {
		return false
	}

	if changed != "0" {
		return false
	}

	return true
}
