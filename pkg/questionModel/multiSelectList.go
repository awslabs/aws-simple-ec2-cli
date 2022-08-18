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
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

/*
MultiSelectList represents a question with a list of options from which multiple options may be chosen as the answer.
Options may be presented in a table based on initialized input.
*/
type MultiSelectList struct {
	list            list.Model      // The list of options
	selected        map[int]item    // Map of selected items in the list
	itemMap         map[item]string // Maps the item chosen to the answer value
	header          string          // The header for the item list table
	question        string          // The question being asked
	err             error           // An error caught during the question
	displayErrorMsg bool            // If the error message should be displayed
	errorMsg        string          // Error msg allerting the user they have to choose an option
}

// InitializeModel initializes the model based on the passed in question input.
func (m *MultiSelectList) InitializeModel(input *QuestionInput) {
	header, items, itemMap := createItems(input)
	items = append(items, item("SUBMIT"))

	// Define how list items are rendered in their focused and unfocused states
	itemDelegate := itemDelegate{
		renderUnfocused: func(s string, index int) string {
			if index == len(items)-1 {
				return fmt.Sprintf(xLargeLeftPadding.Render("\n[ %s ]"), blurred.Render(s))
			}
			return styleTableItemRows(fmt.Sprintf("%s %s", m.getCheckBox(index), s), xLargeLeftPadding, noStyle, mediumLeftPadding)
		},
		renderFocused: func(s string, index int) string {
			if index == len(items)-1 {
				return fmt.Sprintf(xLargeLeftPadding.Render("\n[ %s ]"), focused.Render(s))
			}
			return styleTableItemRows(fmt.Sprintf("> %s %s", m.getCheckBox(index), s), xLargeLeftPadding, focused,
				smallLeftPadding.Copy().Inherit(focused))
		},
	}

	m.list = createModelList(items, itemDelegate, 0)
	m.header = header
	m.itemMap = itemMap
	m.question = input.QuestionString
	m.errorMsg = "Please choose at least one option"

	// Create selected map and select defaults
	m.selected = make(map[int]item)
	defaultIndexes := getDefaultOptionIndexes(input)
	for _, defaultIndex := range defaultIndexes {
		m.selected[defaultIndex] = items[defaultIndex].(item)
	}
}

// Init defines an optional command that can be run when the question is asked.
func (m *MultiSelectList) Init() tea.Cmd {
	return nil
}

/*
Update is called when a message is received. Handles user input to traverse through list,
select answers, and submit selected answers.
*/
func (m *MultiSelectList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.err = exitError
			return m, tea.Quit

		case tea.KeyEnter, tea.KeySpace:
			m.displayErrorMsg = false
			if m.isButtonFocused() {
				if len(m.selected) == 0 {
					m.displayErrorMsg = true
					return m, nil
				}
				return m, tea.Quit
			}
			m.selectItem()
		}

	case error:
		m.err = msg
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the view for the question. The view is rendered after every update
func (m *MultiSelectList) View() string {
	b := strings.Builder{}
	if m.question != "" {
		b.WriteString(m.question + "\n\n")
	}

	if m.displayErrorMsg {
		b.WriteString(mediumLeftPadding.Copy().Inherit(errorStyle).Render(m.errorMsg) + "\n")
	}

	if m.header != "" {
		b.WriteString(xLargeLeftPadding.Render(m.header) + "\n")
	}
	b.WriteString(m.list.View())
	return b.String()
}

// GetSelectedValues gets a list of all of the selected values
func (m *MultiSelectList) GetSelectedValues() []string {
	values := make([]string, 0, len(m.selected))
	for _, value := range m.selected {
		values = append(values, m.itemMap[value])
	}
	return values
}

// getCheckBox gets a checked or unchecked checkbox based on the selection state at the given item index.
func (m *MultiSelectList) getCheckBox(checkBoxIndex int) string {
	checked := "[ ]"
	if _, ok := m.selected[checkBoxIndex]; ok {
		checked = "[x]"
	}
	return checked
}

// selectItem selects the focused item, or unselects the focused item if already selected
func (m *MultiSelectList) selectItem() {
	_, ok := m.selected[m.list.Cursor()]
	if ok {
		delete(m.selected, m.list.Cursor())
	} else {
		i, ok := m.list.SelectedItem().(item)
		if ok {
			m.selected[m.list.Cursor()] = i
		}
	}
}

// getError gets the error from the question if one arose
func (m *MultiSelectList) GetError() error { return m.err }

// isButtonFocused returns if the submit button is focused or not
func (m *MultiSelectList) isButtonFocused() bool { return m.list.Cursor() == len(m.list.Items())-1 }
