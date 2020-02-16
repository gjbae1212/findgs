package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type execCommand func(cmd *cobra.Command, args []string)

var (
	rootCmd = &cobra.Command{
		Use:   "findgs",
		Short: color.GreenString("findgs can search your starred repositories in the Github which matched searching text from title, description, topic, and README."),
		Long:  color.GreenString("findgs can search your starred repositories in the Github which matched searching text from title, description, topic, and README.\nIt's very useful when you have many starred Github Repositories, because for using it in someday."),
	}
)

var (
	personalGithubToken string
)

var (
	ErrNotFoundGithubToken = errors.New("[err] Not Found Github Token, you should pass it by \"GITHUB_TOKEN\" ENV or -t option.")
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("token", "t", "", color.CyanString("Github Token (default is \"GITHUB_TOKEN\" ENV)"))

	// mapping viper.
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))

	// hide help option.
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

func initConfig() {
	token := os.Getenv("GITHUB_TOKEN")
	passedToken := viper.Get("token").(string)
	if passedToken != "" {
		token = passedToken
	}
	personalGithubToken = token
}

func panicError(err error) {
	fmt.Println(color.RedString("%s", err.Error()))
	os.Exit(1)
}
