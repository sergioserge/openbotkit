package learnings

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	learningssvc "github.com/73ai/openbotkit/service/learnings"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all saved learnings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st := learningssvc.New(config.LearningsDir())
		results, err := st.Search(args[0])
		if err != nil {
			return err
		}
		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}
		for _, r := range results {
			fmt.Printf("[%s] %s\n", r.Topic, r.Line)
		}
		return nil
	},
}
