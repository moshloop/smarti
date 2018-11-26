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
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			log.Infof("Running containers on %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())

			var inv = pkg.Parse(cmd)

			for _, container := range inv.Containers() {
				fmt.Printf("%s\n", strings.Replace(container.Image, ":", ": ", 1))
			}
		},
	}
	Containers = cobra.Command{

		Use:  "containers",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			log.Infof("Running containers on %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())

			var inv = pkg.Parse(cmd)

			for _, container := range inv.Containers() {
				fmt.Printf("%s\n", container.ToDeployment())
			}

		},
	}


)


