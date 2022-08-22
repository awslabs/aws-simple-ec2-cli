// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.
package questionModel

import (
	"errors"
	"fmt"
	"io"
	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/ec2helper"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

const (
	defaultWidth       = 20
	columnSeperator    = "│"
	headerSeperator    = "─"
	rowColIntersect    = "┼"
	tableLineMaxLength = 300
)

var (
	// Styling to add left padding to strings
	noStyle           = lipgloss.NewStyle()
	xSmallLeftPadding = lipgloss.NewStyle().PaddingLeft(1)
	smallLeftPadding  = lipgloss.NewStyle().PaddingLeft(3)
	mediumLeftPadding = lipgloss.NewStyle().PaddingLeft(5)
	largeLeftPadding  = lipgloss.NewStyle().PaddingLeft(7)
	xLargeLeftPadding = lipgloss.NewStyle().PaddingLeft(9)

	focused    = lipgloss.NewStyle().Foreground(lipgloss.Color("170")) // Pink
	blurred    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // Red

	boldStyle = lipgloss.NewStyle().Bold(true)
	helpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	exitError = errors.New("Exiting the questionnaire")
)

var yesNoData = [][]string{{cli.ResponseYes}, {cli.ResponseNo}}
var yesNoOptions = []string{cli.ResponseYes, cli.ResponseNo}

// CheckInput is used to validate a given string using validation methods from ec2helper
type CheckInput func(*ec2helper.EC2Helper, string) bool

type Row [][]string

type AskQuestionStruct struct{}

// QuestionInput represents input that can be used to initialize each question
type QuestionInput struct {
	DefaultOption     string               // Defaulted set/selected answer
	DefaultOptionList []string             // List of default selected answers
	Rows              []Row                // Row data used to fill in question tables
	HeaderStrings     []string             // List of headers for question tables
	IndexedOptions    []string             // List of values to be returned when selected index in a list is chosen
	QuestionString    string               // The Question being asked
	EC2Helper         *ec2helper.EC2Helper // EC2Helper to provide validation methods for text inputs
	Fns               []CheckInput         // List of input check functions to validate text inputs
}

/*
questionModel represents a question. Builds on BubbleTea's tea.Model interface to allow
for the initialization of a question model and to retrieve any errors that may occur
*/
type QuestionModel interface {
	InitializeModel(input *QuestionInput)
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	GetError() error
}

// item represents an item, or row, in a list
type item string

// FilterValue is the value used when filtering against the item in a list.
// Used to implement the list.Item iterface
func (i item) FilterValue() string { return "" }

// itemDelegate defines how an item is rendered in a list
type itemDelegate struct {
	renderUnfocused func(str string, index int) string
	renderFocused   func(str string, index int) string
}

// Methods needed implement the itemDelegate interface
func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

// Render renders an item, or row,  in a list. Also needed to implement the itemDelegate interface
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := d.renderUnfocused(string(i), index)
	if index == m.Index() {
		str = d.renderFocused(string(i), index)
	}

	fmt.Fprintf(w, str)
}

/*
AskQuestion initializes the given question model with question input and asks the question. Finishes
when answer is given, or user exits out of the question. Returns the error from the question
model.
*/
func (a *AskQuestionStruct) AskQuestion(model QuestionModel, questionInput *QuestionInput) error {
	fmt.Println()
	model.InitializeModel(questionInput)
	p := tea.NewProgram(model)
	err := p.Start()
	if model.GetError() != nil {
		err = model.GetError()
	}
	return err
}

// combineRows combines all of the rows into a 2d-array
func combineRows(rows []Row) [][]string {
	stringRows := [][]string{}
	for _, row := range rows {
		stringRows = append(stringRows, row...)
	}
	return stringRows
}

/*
createItems creates the items for a list in a question. The items are made from a question table along with
the table's header, and a map to retrieve indexed answers.
*/
func createItems(input *QuestionInput) (header string, itemList []list.Item, itemMap map[item]string) {
	tableString := createQuestionTable(combineRows(input.Rows), input.HeaderStrings)
	optionStrings := strings.Split(strings.TrimSuffix(tableString, "\n"), "\n")

	// Remove Empty Lines
	for index := 0; index < len(optionStrings); index++ {
		optionString := optionStrings[index]
		if strings.TrimSpace(optionString) == "" {
			optionStrings = append(optionStrings[0:index], optionStrings[index+1:]...)
			index--
		}
	}

	// Seperate the header from the table rows
	header = ""
	if len(input.HeaderStrings) != 0 && len(optionStrings) > 0 {
		header = createHeader(optionStrings)
		optionStrings = optionStrings[1:]
	}

	// Creates list of items and item map
	itemList = []list.Item{}
	itemMap = map[item]string{}
	index := 0
	b := strings.Builder{}
	if input.Rows != nil {
		// Create an item for each Row
		for i, row := range input.Rows {
			b.Reset()
			// Combine lines to form the Row
			for i := 0; i < len(row); i++ {
				b.WriteString(optionStrings[index])
				if len(row)-i > 1 {
					b.WriteRune('\n')
				}
				index++
			}
			itemString := b.String()
			if i < len(input.IndexedOptions) {
				itemMap[item(itemString)] = input.IndexedOptions[i]
			}
			if strings.TrimSpace(itemString) != "" {
				itemList = append(itemList, item(itemString))
			}
		}
	}

	return header, itemList, itemMap
}

// createHeader creates a formatted table header
func createHeader(optionStrings []string) string {
	headers := optionStrings[0]
	if len(optionStrings) == 1 {
		return headers
	}

	rowEntries := strings.Split(optionStrings[1], columnSeperator)
	b := &strings.Builder{}
	b.WriteString(styleTableItem(headers, boldStyle, boldStyle) + "\n")
	for index, entry := range rowEntries {
		b.WriteString(strings.Repeat(headerSeperator, len(entry)))
		if index != len(rowEntries)-1 {
			b.WriteString(rowColIntersect)
		}
	}
	return b.String()
}

/*
createModelList creates a model list to be used in a list type question. Sets the initial selected option as
the given default option.
*/
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

// stringToInterface converts a list of strings to a list of interfaces
func stringToInterface(s []string) []interface{} {
	result := make([]interface{}, len(s))
	for i, str := range s {
		result[i] = str
	}
	return result
}

/*
createQuestionTable creates a table to have a formatted display for options in questions.
*/
func createQuestionTable(tableData [][]string, headers []string) string {
	// Fill in missing table data
	numColumns := 0
	for _, str := range tableData {
		if len(str) > numColumns {
			numColumns = len(str)
		}
	}
	for index := range tableData {
		for i := 0; len(tableData[index]) < numColumns; i++ {
			tableData[index] = append(tableData[index], "")
		}
	}

	tableBuilder := &strings.Builder{}
	tableWriter := tablewriter.NewWriter(tableBuilder)
	tableWriter.SetHeader(headers)
	tableWriter.SetAutoWrapText(false)
	tableWriter.SetAutoFormatHeaders(true)
	tableWriter.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tableWriter.SetAlignment(tablewriter.ALIGN_LEFT)
	tableWriter.SetColumnSeparator(columnSeperator)
	tableWriter.SetRowSeparator("")
	tableWriter.SetHeaderLine(false)
	tableWriter.SetBorder(false)
	tableWriter.SetTablePadding("")
	tableWriter.AppendBulk(tableData)
	tableWriter.Render()

	tableString := tableBuilder.String()
	return tableString
}

// styleTableItemRows styles multiple items in a single row
func styleTableItemRows(tableItemRow string, multiRowStyle lipgloss.Style, defaultStyle lipgloss.Style,
	firstRowStyle lipgloss.Style) string {
	b := strings.Builder{}
	splitStrings := strings.Split(tableItemRow, "\n")
	for index, str := range splitStrings {
		if index == 0 {
			b.WriteString(styleTableItem(str, defaultStyle, firstRowStyle))
		} else {
			b.WriteString(styleTableItem(str, defaultStyle, multiRowStyle.Copy().Inherit(defaultStyle)))
		}
		if index < len(splitStrings)-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

/*
styleTableItem applies a lipgloss style to each string in a table item, negelecting the column seperator.
The style for the first column in the table is specified seperately. If the first column is to be the same as
the rest, then set the same value for the style and firstColumnStyle parameters.
*/
func styleTableItem(tableItem string, style lipgloss.Style, firstColumnStyle lipgloss.Style) string {
	splitString := strings.Split(tableItem, columnSeperator)
	for i := 0; i < len(splitString); i++ {
		if i == 0 {
			splitString[i] = firstColumnStyle.Copy().Inherit(firstColumnStyle).Render(splitString[i])
		} else {
			splitString[i] = style.Render(splitString[i])
		}
	}
	return strings.Join(splitString, columnSeperator)
}

// getDefaultOptionIndex gets the index of the default option. If not found then -1 is returned
func getDefaultOptionIndex(input *QuestionInput) int {
	defaultOptionIndex := -1
	for index, option := range input.IndexedOptions {
		if option == input.DefaultOption {
			defaultOptionIndex = index
			break
		}
	}
	return defaultOptionIndex
}

// getDefaultOptionIndexes gets a list of indexes of default options
func getDefaultOptionIndexes(input *QuestionInput) []int {
	defaultOptionIndexes := []int{}
	for _, option := range input.DefaultOptionList {
		input.DefaultOption = option
		if defaultOptionIndex := getDefaultOptionIndex(input); defaultOptionIndex != -1 {
			defaultOptionIndexes = append(defaultOptionIndexes, defaultOptionIndex)
		}
	}
	return defaultOptionIndexes
}

// CreateSingleLineRows converts a 2d-array into a list of rows
func CreateSingleLineRows(data [][]string) []Row {
	rows := []Row{}
	for _, str := range data {
		rows = append(rows, [][]string{str})
	}
	return rows
}

// askYesNoQuestion asks a yes or no question
func AskYesNoQuestion(qh *QuestionModelHelper, question string,
	defaultToYes bool) (string, error) {
	defaultOption := cli.ResponseNo
	if defaultToYes {
		defaultOption = cli.ResponseYes
	}

	model := &SingleSelectList{}
	err := qh.Svc.AskQuestion(model, &QuestionInput{
		QuestionString: question,
		IndexedOptions: yesNoOptions,
		DefaultOption:  defaultOption,
		Rows:           CreateSingleLineRows(yesNoData),
	})

	if err != nil {
		return "", err
	}

	return model.GetChoice(), nil
}
