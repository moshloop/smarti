package cmd

import (
	"github.com/spf13/cobra"
	"github.com/moshloop/smarti/pkg"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

var (

	Versions = cobra.Command{
		Use:  "versions",
		Short: "Print a list of all images and their resolved image tags",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			log.Infof("Exporting images versions from %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())

			var inv = pkg.Parse(cmd)

			for _, container := range inv.Containers() {
				fmt.Printf("%s\n", strings.Replace(container.Image, ":", ": ", 1))
			}
		},
	}

	Spec = cobra.Command{

		Use:  "spec",
		Short: "Generate a kubernetes spec for deployment",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			log.Infof("Running containers on %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())

			var inv = pkg.Parse(cmd)

			for _, container := range inv.Containers() {
				fmt.Printf("%s\n", container.ToDeployment())
			}

		},
	}

	Containers = cobra.Command{

		Use:  "k8s",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			print("Must specify a sub-command, run `smarti k8s --help` for more details")
		},


	}


)


