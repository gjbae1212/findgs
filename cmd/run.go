package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
	"github.com/gjbae1212/findgs/search"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
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
	minScore = float64(0.1)
)

var (
	searchSuggest = prompt.Suggest{Text: "search", Description: "Search starred github repositories which matched text from Readme, description, topic, name ... and so on."}
	exitSuggest   = prompt.Suggest{Text: "exit", Description: "Good bye."}
	openSuggest   = prompt.Suggest{Text: "open", Description: "Open a selected repository of found repositories to browser."}
	listSuggest   = prompt.Suggest{Text: "list", Description: "Show searched repositories recently through search command."}
	scoreSuggest  = prompt.Suggest{Text: "score", Description: "Set the score that can search repositories equal to or higher than the score.( 0 <= score)"}

	openNumSuggest  = prompt.Suggest{Text: "num", Description: "Open url to browser using num value."}
	openNameSuggest = prompt.Suggest{Text: "name", Description: "Open url to browser using name value."}
)

var (
	foundList             []*search.Result
	foundMap              map[string]*search.Result
	recentlySearchKeyword string
)

var (
	pt *prompt.Prompt
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
	}
}

func run() execCommand {
	return func(cmd *cobra.Command, args []string) {
		fmt.Println("\033[2J")
		fmt.Print("\033[H")
		total, _ := searcher.TotalDoc()
		color.Green("Indexing %d", total)
		color.Green("Searching repositories equal to or higher %.3f score", minScore)
		fmt.Println()
		pt = prompt.New(executor, completer,
			prompt.OptionPrefix(fmt.Sprintf("(Score: %.3f) >> ", minScore)),
			prompt.OptionPrefixTextColor(prompt.Cyan),
			prompt.OptionShowCompletionAtStart())
		pt.Run()
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	text := strings.ToLower(strings.TrimSpace(d.Text))

	suggests := []prompt.Suggest{}
	switch {
	case text == "":
		suggests = append(suggests, searchSuggest, openSuggest, listSuggest, scoreSuggest, exitSuggest)
	case "exit" != text && strings.Contains("exit", text):
		suggests = append(suggests, exitSuggest)
	case "open" != text && strings.Contains("open", text):
		suggests = append(suggests, openSuggest)
	case "search" != text && strings.Contains("search", text):
		suggests = append(suggests, searchSuggest)
		fallthrough
	case "score" != text && strings.Contains("score", text):
		suggests = append(suggests, scoreSuggest)
	case "list" != text && strings.Contains("list", text):
		suggests = append(suggests, listSuggest)
	case strings.HasPrefix(text, "score"):
		if text == "score" {
			for i := 0; i < 10; i++ {
				suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("0.%d", i)})
			}
			break
		}
		subText := strings.TrimSpace(text[5:])

		if subText == "0" || subText == "0." {
			for i := 0; i < 10; i++ {
				suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("0.%d", i)})
			}
			break
		}

		if _, err := strconv.ParseFloat(subText, 64); err == nil {
			for i := 0; i < 10; i++ {
				suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("%s%d", subText, i)})
			}
			break
		}
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
			break
		}

		if strings.HasPrefix(subText, "num") {
			seps := strings.Split(subText, " ")
			if len(seps) > 1 {
				numText := strings.TrimSpace(strings.Join(seps[1:], " "))
				for i, _ := range foundList {
					if strings.HasPrefix(fmt.Sprintf("%d", i+1), numText) {
						suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("%d", i+1)})
					}
				}
				break
			}
		}

		if strings.HasPrefix(subText, "name") {
			seps := strings.Split(subText, " ")
			if len(seps) > 1 {
				repositoryText := strings.TrimSpace(strings.Join(seps[1:], " "))
				for _, f := range foundList {
					if strings.HasPrefix(strings.ToLower(f.FullName), repositoryText) {
						suggests = append(suggests, prompt.Suggest{Text: fmt.Sprintf("%s", f.FullName)})
					}
				}
				break
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
	case "score":
		scoreText := strings.TrimSpace(strings.Join(seps[1:], " "))
		if f, err := strconv.ParseFloat(scoreText, 64); err != nil {
			color.Red("Wrong score %s", scoreText)
		} else {
			if f < 0 {
				color.Red("Required 0 <= score")
				break
			}
			minScore = f
			opt := prompt.OptionPrefix(fmt.Sprintf("(Score: %.3f) >> ", minScore))
			opt(pt)
			color.Green("Set score %.3f", minScore)
		}
	case "search":
		recentlySearchKeyword = strings.Join(seps[1:], " ")
		result, err := searcher.Search(recentlySearchKeyword, minScore)
		if err != nil {
			color.Red("%s", err)
			return
		}

		foundList = []*search.Result{}
		foundMap = make(map[string]*search.Result)
		for _, found := range result {
			if found.Score >= minScore {
				foundMap[strings.ToLower(found.FullName)] = found
				foundList = append(foundList, found)
			}
		}
		showSearchedList()
	default:
		color.Red("Not Found Command.")
	}
}

func showSearchedList() {
	// clear terminal.
	fmt.Println("\033[2J")
	fmt.Print("\033[H")
	color.Green("[search][text] \"%s\"", recentlySearchKeyword)
	fmt.Println()

	// table writer
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NUM", "SCORE", "NAME", "URL", "TOPIC", "DESCRIPTION"})
	table.SetFooter([]string{"", "", "", "", "TOTAL", fmt.Sprintf("%d", len(foundList))})
	table.SetBorder(false)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetRowLine(true)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgRedColor})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{tablewriter.Bold})
	table.SetFooterColor(
		tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgRedColor, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.BgGreenColor, tablewriter.FgHiWhiteColor})

	data := [][]string{}
	for i, found := range foundList {
		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%f", found.Score),
			found.FullName,
			found.Url,
			fmt.Sprintf("%s", found.Topics),
			found.Description,
		})
	}
	table.AppendBulk(data)
	table.Render()
}

func init() {
	rootCmd.AddCommand(runCommand)
}
