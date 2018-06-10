package cmd



import "github.com/spf13/cobra"

var (
	Group = cobra.Command{
		Use:  "each-group",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	Host = cobra.Command{
		Use:  "each-host",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
)


