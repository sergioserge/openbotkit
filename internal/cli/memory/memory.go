package memory

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage personal memory",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(deleteCmd)
}
