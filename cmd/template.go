package cmd

import (
"github.com/spf13/cobra"
"github.com/moshloop/smarti/pkg"
"github.com/json-iterator/go"
"fmt"
)

var (
	Template = cobra.Command{
		Use:  "template",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			var inv = pkg.Parse(cmd)

			out := make(map[string]interface{})

			for _, group := range inv.Groups {
				out[group.Name] = group.Vars
			}

			var json = jsoniter.ConfigCompatibleWithStandardLibrary

			data, err := json.Marshal(out)

			if err != nil {
				fmt.Println("error:", err)
			} else {
				fmt.Printf("%s\n", data)
			}
		},
	}
)

