package history

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "history",
	Short: "Manage conversation history",
}

func init() {
	Cmd.AddCommand(captureCmd)
}
