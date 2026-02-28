package memory

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage memory data source",
}

func init() {
	Cmd.AddCommand(captureCmd)
}
