package cmd

import (
	"github.com/gjbae1212/findgs/search"
	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	clearCommand = &cobra.Command{
		Use:    "clear",
		Short:  color.YellowString("Clear all of cached data such as indexing, database."),
		Long:   color.YellowString("Clear all of cached data. It's to clear cached data such as indexing, a database made for the sake of high performance, limited Github API Quota."),
		PreRun: preClear(),
		Run:    clear(),
	}
)

func preClear() execCommand {
	return func(cmd *cobra.Command, args []string) {
		if personalGithubToken == "" {
			panicError(ErrNotFoundGithubToken)
		}
	}
}

func clear() execCommand {
	return func(cmd *cobra.Command, args []string) {
		result := false
		prompt := &survey.Confirm{
			Message: "Do you want clear all of cached data?",
		}
		survey.AskOne(prompt, &result)
		if result {
			if err := search.ClearAll(); err != nil {
				panicError(err)
			} else {
				color.Green("[success] clear all of cached data.")
			}
		} else {
			color.Yellow("[cancel] clear all of cached data.")
		}
	}
}

func init() {
	rootCmd.AddCommand(clearCommand)
}
