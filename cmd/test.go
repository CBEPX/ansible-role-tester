// Copyright © 2018 Karl Hepworth Karl.Hepworth@gmail.com
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fubarhouse/ansible-role-tester/util"
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"strings"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Execute tests against an existing container",
	Long: `Execute tests against an existing container

If container does not exist it will be created, however
containers won't be removed after completion.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := util.AnsibleConfig{
			HostPath:         source,
			Inventory:		  inventory,
			RemotePath:       destination,
			RequirementsFile: requirements,
			PlaybookFile:     playbook,
			Verbose:          verbose,
			Quiet:			  quiet,
		}

		dist, _ := util.GetDistribution(image, image, "/sbin/init", "/sys/fs/cgroup:/sys/fs/cgroup:ro", user, distro)

		dist.CID = containerID

		if dist.DockerCheck() {

			if inventory != "" {
				invfile := fmt.Sprintf(source + "/" + inventory)
				if _, err := os.Stat(invfile); os.IsNotExist(err) {
					if !quiet {
						log.Fatalf("Specified inventory file %v does not exist.", invfile)
						log.Println(invfile)
					}
				}
			}

			// Adjust playbook path
			if remote {
				// The playbook will be located on the host if the remote flag is enabled.
				if strings.HasPrefix(config.PlaybookFile, "./") {
					pwd, _ := os.Getwd()
					config.PlaybookFile = fmt.Sprintf("%v/%v", pwd, config.PlaybookFile)
				} else if strings.HasPrefix(config.PlaybookFile, "/") {
					config.PlaybookFile = fmt.Sprintf("%v", config.PlaybookFile)
				} else if !remote {
					config.PlaybookFile = fmt.Sprintf("%v/tests/%v", source, config.PlaybookFile)
				} else if remote {
					config.PlaybookFile = fmt.Sprintf("%v/tests/%v", source, config.PlaybookFile)
				}
				fp := fmt.Sprintf(config.PlaybookFile)
				if _, err := os.Stat(fp); os.IsNotExist(err) {
					if !quiet {
						log.Fatalf("Specified playbook file %v does not exist.", fp)
					}
				}
			} else {
				// The playbook will be located on the container (via mount) if the remote flag is enabled.
				config.PlaybookFile = fmt.Sprintf("/etc/ansible/roles/role_under_test/%v", config.PlaybookFile)
				pwd, _ := os.Getwd()
				file := fmt.Sprintf("%v/%v", pwd, playbook)
				fp := fmt.Sprintf(file)
				if _, err := os.Stat(fp); os.IsNotExist(err) {
					if !quiet {
						log.Fatalf("Specified playbook file %v does not exist.", fp)
					}
				}
			}

			if !remote {
				dist.RoleSyntaxCheck(&config)
				dist.RoleTest(&config)
				dist.IdempotenceTest(&config)
			} else {
				dist.RoleSyntaxCheckRemote(&config)
				dist.RoleTestRemote(&config)
				dist.IdempotenceTestRemote(&config)
			}
		} else {
			if !quiet {
				log.Warnf("Container %v is not currently running", dist.CID)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	pwd, _ := os.Getwd()
	testCmd.Flags().StringVarP(&containerID, "name", "n", containerID, "Container ID")
	testCmd.Flags().StringVarP(&inventory, "inventory", "e", "", "Inventory file")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose mode for Ansible commands.")
	testCmd.Flags().StringVarP(&playbook, "playbook", "p", "playbook.yml", "The filename of the playbook")
	testCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Enable quiet mode")
	testCmd.Flags().StringVarP(&source, "source", "s", pwd, "Location of the role to test")
	testCmd.Flags().BoolVarP(&remote, "remote", "m", false, "Run the test remotely to the container")

	testCmd.MarkFlagRequired("name")
}
