package questionModel

import (
	"fmt"
	"io"
	"os"
	"simple-ec2/pkg/ec2helper"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

const defaultWidth = 20

var (
	noStyle           = lipgloss.NewStyle()
	xSmallLeftPadding = lipgloss.NewStyle().PaddingLeft(1)
	smallLeftPadding  = lipgloss.NewStyle().PaddingLeft(3)
	mediumLeftPadding = lipgloss.NewStyle().PaddingLeft(5)
	largeLeftPadding  = lipgloss.NewStyle().PaddingLeft(7)
	xLargeLeftPadding = lipgloss.NewStyle().PaddingLeft(9)
	focused           = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	blurred           = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

type CheckInput func(*ec2helper.EC2Helper, string) bool

type BubbleTeaData struct {
	DefaultOption     string
	DefaultOptionList []string
	OptionData        [][]string
	HeaderStrings     []string
	IndexedOptions    []string
	QuestionString    string
	EC2Helper         *ec2helper.EC2Helper
	Fns               []CheckInput
}

type bubbleModel interface {
	InitializeModel(data *BubbleTeaData)
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	getError() error
}

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct {
	renderUnselected func(str string, index int) string
	renderSelected   func(str string, index int) string
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := d.renderUnselected(string(i), index)
	if index == m.Index() {
		str = d.renderSelected(string(i), index)
	}

	fmt.Fprintf(w, str)
}

func AskQuestion(model bubbleModel, questionData *BubbleTeaData) error {
	fmt.Println()
	model.InitializeModel(questionData)
	p := tea.NewProgram(model)
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	return model.getError()
}

func createItems(data *BubbleTeaData) (header string, itemList []list.Item, itemMap map[item]string) {
	tableString := createQuestionTable(data.OptionData, data.HeaderStrings)
	optionStrings := strings.Split(strings.TrimSuffix(tableString, "\n"), "\n")

	// Remove Empty Lines
	for index := 0; index < len(optionStrings); index++ {
		optionString := optionStrings[index]
		if strings.TrimSpace(optionString) == "" {
			optionStrings = append(optionStrings[0:index], optionStrings[index+1:]...)
			index--
		}
	}

	header = ""
	if len(data.HeaderStrings) != 0 {
		header = strings.Join(optionStrings[0:2], "\n")
		optionStrings = optionStrings[2:]
	}

	itemList = []list.Item{}
	itemMap = make(map[item]string, len(data.OptionData))
	for index, itemString := range optionStrings {
		if len(data.IndexedOptions) == len(data.OptionData) {
			itemMap[item(itemString)] = data.IndexedOptions[index]
		}
		if strings.TrimSpace(itemString) != "" {
			itemList = append(itemList, item(itemString))
		}

	}

	return header, itemList, itemMap
}

func createModelList(items []list.Item, itemDelegate itemDelegate, defaultOptionIndex int) list.Model {
	modelList := list.New(items, itemDelegate, defaultWidth, len(items)+1)
	modelList.SetShowStatusBar(false)
	modelList.SetFilteringEnabled(false)
	modelList.SetShowTitle(false)
	modelList.Styles.HelpStyle = helpStyle
	modelList.SetShowPagination(false)
	modelList.Select(defaultOptionIndex)
	modelList.DisableQuitKeybindings()
	modelList.SetShowHelp(false)
	return modelList
}

func createQuestionTable(tableData [][]string, headers []string) string {
	if len(headers) != 0 {
		headerSeperators := []string{}
		for index := 0; index < len(headers); index++ {
			headerSeperators = append(headerSeperators, "---")
		}
		tableData = append([][]string{headerSeperators}, tableData...)
	}

	tableBuilder := &strings.Builder{}
	table := tablewriter.NewWriter(tableBuilder)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("|")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("")   // pad with tabs
	table.AppendBulk(tableData) // Add Bulk Data
	table.Render()

	tableString := tableBuilder.String()

	if len(headers) != 0 {
		renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("notty"))
		if err == nil {
			tableString, err = renderer.Render(tableString)
		}
	}

	return tableString
}

func focusTableItem(tableItem string) string {
	splitString := strings.Split(tableItem, "│")
	for i := 0; i < len(splitString); i++ {
		if i == 0 {
			splitString[i] = smallLeftPadding.Copy().Inherit(focused).Render(splitString[i])
		} else {
			splitString[i] = focused.Render(splitString[i])
		}
	}
	return strings.Join(splitString, "│")
}

func getDefaultOptionIndex(data *BubbleTeaData) int {
	defaultOptionIndex := -1
	for index, option := range data.IndexedOptions {
		if option == data.DefaultOption {
			defaultOptionIndex = index
			break
		}
	}
	return defaultOptionIndex
}

func getDefaultOptionIndexes(data *BubbleTeaData) []int {
	defaultOptionIndexes := []int{}
	for _, option := range data.DefaultOptionList {
		data.DefaultOption = option
		if defaultOptionIndex := getDefaultOptionIndex(data); defaultOptionIndex != -1 {
			defaultOptionIndexes = append(defaultOptionIndexes, defaultOptionIndex)
		}
	}
	return defaultOptionIndexes
}
