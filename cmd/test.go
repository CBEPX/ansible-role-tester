package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Execute tests against an existing container",
	Long: `Execute tests against an existing container

If container does not exist it will be created, however
containers won't be removed after completion.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := AnsibleConfig{
			source,
			destination,
			requirements,
			playbook,
		}

		e, dist := getDistribution(image, image, "/sbin/init", "/sys/fs/cgroup:/sys/fs/cgroup:ro")
		if e != nil {
			logrus.Errorln("Incompatible disribution was inputted, attempting autofix.")
		}

		dist.install(&config)
		dist.test_syntax(&config)
		dist.test_role(&config)
		dist.test_idempotence(&config)
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	pwd, _ := os.Getwd()
	testCmd.Flags().StringVarP(&containerID, "id", "n", containerID, "Container ID")
	testCmd.Flags().StringVarP(&source, "source", "s", pwd, "Location of the role to test")
	testCmd.Flags().StringVarP(&destination, "destination", "d", "/etc/ansible/roles/role_under_test", "Location which the role will be mounted to")
	testCmd.Flags().StringVarP(&requirements, "requirements", "r", "", "Path to requirements file.")
	testCmd.Flags().StringVarP(&playbook, "playbook", "p", "playbook.yml", "The filename of the playbook")
	testCmd.Flags().StringVarP(&image, "image", "i", "fubarhouse/docker-ansible:bionic", "The image reference to use.")
}
