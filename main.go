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
			case level > 2:
				log.SetLevel(log.DebugLevel)
			case level > 1:
				log.SetLevel(log.InfoLevel)
			case level > 0:
				log.SetLevel(log.WarnLevel)
			default:
				log.SetLevel(log.ErrorLevel)
			}
		},

	}

	root.PersistentFlags().StringP("inventory", "i", "", "specify inventory host path or comma separated host list")
	root.PersistentFlags().Bool("version", false, "")
	root.PersistentFlags().StringSliceP("extra-vars", "e", []string{}, " set additional variables as key=value or YAML/JSON, if filename prepend with @")
	root.PersistentFlags().StringP("limit", "l", "", "further limit selected hosts to an additional pattern")
	root.PersistentFlags().CountP("loglevel", "v", "Increase logging level")

	root.AddCommand(&cmd.List, &cmd.Group, &cmd.Host, &cmd.Template)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}

}
