package util

import (
	"strings"

	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strconv"
	"time"
	"os"
)

// IdempotenceTest will run an Ansible playbook once and check the
// output for any changed or failed tasks as reported by Ansible.
func (dist *Distribution) IdempotenceTest(config *AnsibleConfig) {

	// Test role idempotence.
	if !config.Quiet {
		log.Infoln("Testing role idempotence...")
	}

	args := []string{
		"exec",
		"--tty",
		dist.CID,
		"ansible-playbook",
		fmt.Sprintf("%v/tests/%v", config.RemotePath, config.PlaybookFile),
	}

	// Add verbose if configured
	if config.Verbose {
		args = append(args, "-vvvv")
	}

	now := time.Now()
	var idempotence = false
	if !config.Quiet {
		out, _ := DockerExec(args, true)
		idempotence = IdempotenceResult(out)
	} else {
		out, _ := DockerExec(args, false)
		idempotence = IdempotenceResult(out)
	}

	if !config.Quiet {
		log.Infof("Idempotence was checked in %v", time.Since(now))
		if idempotence {
			log.Infoln("Idempotence test: PASS")
		} else {
			log.Errorln("Idempotence test: FAIL")
			os.Exit(1)
		}
	}
}

// IdempotenceResult will get the result of an idempotence test
// which is the full output of a role, and it will identify each
// of the applicable checks for idempotence. In this case, we
// simply need the values of changed and failed and some basic logic.
func IdempotenceResult(output string) bool {

	lines := strings.Split(output, "\n")

	var changed int64
	var failed int64
	error := errors.New("empty error")

	for _, line := range lines {
		if strings.Contains(line, "ok=") && strings.Contains(line, "changed=") {
			f := strings.Split(line, "=")
			if strings.Contains(line, "changed=") {
				changed, error = strconv.ParseInt(strings.Split(f[2], " ")[0], 0, 0)
			}
			if strings.Contains(line, "failed=") {
				failed, error = strconv.ParseInt(strings.Split(f[4], " ")[0], 0, 0)
			}
		}
	}

	if error != nil {
		log.Errorln(error)
		return false
		os.Exit(1)
	}

	if failed > 0 {
		return false
		os.Exit(1)
	}

	if changed > 0 {
		return false
		os.Exit(1)
	}

	return true
}
