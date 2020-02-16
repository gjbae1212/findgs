package cmd

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/gjbae1212/findgs/search"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	runCommand = &cobra.Command{
		Use:    "run",
		Short:  color.YellowString("Run an interactive CLI for searching Starred Github Repositories."),
		Long:   color.YellowString("Run an interactive CLI for searching Starred Github Repositories."),
		PreRun: preRun(),
		Run:    run(),
	}
)

var (
	searcher search.Searcher
	maxSize  int
)

var (
	searchSuggest = prompt.Suggest{Text: "search", Description: "Search starred github repositories which matched text from Readme, topic, name ... and so on."}
	exitSuggest   = prompt.Suggest{Text: "exit", Description: "Good bye."}
	openSuggest   = prompt.Suggest{Text: "open", Description: "Open a selected repository of found repositories to browser."}
	listSuggest   = prompt.Suggest{Text: "list", Description: "Show searched repositories currently."}

	openNumSuggest  = prompt.Suggest{Text: "num", Description: "Open url to browser using num value."}
	openNameSuggest = prompt.Suggest{Text: "name", Description: "Open url to browser using name value."}
)

var (
	foundList []*search.Result
	foundMap  map[string]*search.Result
)

func preRun() execCommand {
	return func(cmd *cobra.Command, args []string) {
		if personalGithubToken == "" {
			panicError(ErrNotFoundGithubToken)
		}
		var err error
		searcher, err = search.NewSearcher(personalGithubToken)
		if err != nil {
			panicError(err)
		}
		if err := searcher.CreateIndex(); err != nil {
			panicError(err)
		}
		maxSize = viper.Get("size").(int)
	}
}

func run() execCommand {
	return func(cmd *cobra.Command, args []string) {
		p := prompt.New(executor, completer,
			prompt.OptionPrefix(">>> "),
			prompt.OptionPrefixTextColor(prompt.DefaultColor),
			prompt.OptionShowCompletionAtStart())
		p.Run()
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	text := strings.ToLower(strings.TrimSpace(d.Text))

	suggests := []prompt.Suggest{}
	switch {
	case text == "":
		suggests = append(suggests, searchSuggest, exitSuggest, openSuggest, listSuggest)
	case "exit" != text && strings.Contains("exit", text):
		suggests = append(suggests, exitSuggest)
	case "open" != text && strings.Contains("open", text):
		suggests = append(suggests, openSuggest)
	case "search" != text && strings.Contains("search", text):
		suggests = append(suggests, searchSuggest)
	case "list" != text && strings.Contains("list", text):
		suggests = append(suggests, listSuggest)
	case strings.HasPrefix(text, "open"):
		if text == "open" {
			suggests = append(suggests, openNumSuggest, openNameSuggest)
			break
		}
		subText := strings.TrimSpace(text[4:])
		if subText != "num" && strings.Contains("num", subText) {
			suggests = append(suggests, openNumSuggest)
			break
		} else if subText != "name" && strings.Contains("name", subText) {
			suggests = append(suggests, openNameSuggest)
			break
		}

		if subText == "num" {
			for i, _ := range foundList {
				suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("%d", i+1)})
			}
			break
		} else if subText == "name" {
			for _, f := range foundList {
				suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("%s", f.FullName)})
			}
		}
	}
	return prompt.FilterHasPrefix(suggests, d.GetWordBeforeCursor(), true)
}

func executor(t string) {
	text := strings.TrimSpace(t)
	seps := strings.Split(text, " ")
	cmd := strings.ToLower(seps[0])
	switch cmd {
	case "exit":
		color.Green("Good Bye.")
		os.Exit(0)
	case "list":
		showSearchedList()
	case "open":
		openText := strings.TrimSpace(strings.Join(seps[1:], " "))
		subSep := strings.Split(openText, " ")
		if len(subSep) == 1 {
			color.Green("Not matched Repository")
			return
		}
		searchText := strings.ToLower(strings.TrimSpace(strings.Join(subSep[1:], " ")))
		if ix, err := strconv.Atoi(searchText); err == nil {
			if ix <= 0 || len(foundList) <= 0 || len(foundList) <= (ix-1) {
				color.Green("Not matched Repository")
				return
			}
			browser.OpenURL(foundList[ix-1].Url)
			return
		} else {
			if repo, ok := foundMap[searchText]; ok {
				browser.OpenURL(repo.Url)
				return
			}
			color.Green("Not matched Repository")
			return
		}
	case "search":
		searchText := strings.Join(seps[1:], " ")
		result, err := searcher.Search(searchText, maxSize)
		if err != nil {
			color.Red("%s", err)
			return
		}
		foundList = result
		foundMap = make(map[string]*search.Result)
		for _, found := range foundList {
			foundMap[found.FullName] = found
		}
		showSearchedList()
	}
}

func showSearchedList() {
	for i, found := range foundList {
		sepFmt := color.BlueString("||")
		fmt.Printf("%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
			color.MagentaString("[num]"), color.GreenString("%d", i+1), sepFmt,
			color.YellowString("[score]"), color.WhiteString("%f", found.Score), sepFmt,
			color.YellowString("[name]"), color.GreenString(found.FullName), sepFmt,
			color.YellowString("[url]"), color.WhiteString(found.Url), sepFmt,
			color.YellowString("[topic]"), color.WhiteString("%s", found.Topics), sepFmt,
			color.YellowString("[desc]"), color.CyanString(found.Description),
		)
		fmt.Println()
	}
}

func init() {
	runCommand.Flags().IntP("size", "s", 100, color.CyanString("max search total count"))

	viper.BindPFlag("size", runCommand.Flags().Lookup("size"))

	rootCmd.AddCommand(runCommand)
}
