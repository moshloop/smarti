package main

import (
	"github.com/spf13/cobra"
	"github.com/moshloop/smarti/cmd"
	"os"
	log "github.com/sirupsen/logrus"
)

func main() {

	var root = &cobra.Command{
		Use: "smarti",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level, _ := cmd.Flags().GetCount("loglevel")
			switch {
			case level > 1:
				log.SetLevel(log.DebugLevel)
			case level > 0:
				log.SetLevel(log.InfoLevel)
			default:
				log.SetLevel(log.WarnLevel)
			}
		},

	}

	root.PersistentFlags().StringP("inventory", "i", "", "Specify inventory host path or comma separated host list")
	root.PersistentFlags().Bool("version", false, "")
	root.PersistentFlags().StringSliceP("extra-vars", "e", []string{}, "Set additional variables as key=value or YAML/JSON, if filename prepend with @")
	root.PersistentFlags().StringP("limit", "l", "", "Limit selected hosts to an additional pattern")
	root.PersistentFlags().CountP("loglevel", "v", "Increase logging level")

	cmd.Containers.AddCommand(&cmd.Versions)
	cmd.Containers.AddCommand(&cmd.Spec)
	cmd.Containers.AddCommand(&cmd.Health)
	cmd.Health.Flags().Bool("print", false, "Print IP:PORT details for running services (useful to pipe into xargs for additional checks)")
	cmd.Containers.PersistentFlags().String("image-versions", "", "A path to yml or json file containing image versions")
	root.AddCommand(&cmd.List, &cmd.Containers)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}

}
