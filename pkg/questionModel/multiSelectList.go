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
	Represents a question with a list of options from which multiple options may be chosen as the answer.
	Options may be presented in a table based on initialized data.
*/
type MultiSelectList struct {
	list     list.Model      // The list of options
	selected map[int]item    // Map of selected items in the list
	itemMap  map[item]string // Maps the item chosen to the answer value
	header   string          // The header for the item list table
	question string          // The question being asked
	err      error           // An error caught during the question
}

// Initializes the model based on the passed in question data
func (m *MultiSelectList) InitializeModel(data *BubbleTeaData) {
	header, items, itemMap := createItems(data)
	items = append(items, item("\n[ Submit ]"))

	// Define how list items are rendered in their focused and unfocused states
	itemDelegate := itemDelegate{
		renderUnfocused: func(s string, index int) string {
			if index == len(items)-1 {
				return xLargeLeftPadding.Copy().Inherit(blurred).Render(s)
			}
			return mediumLeftPadding.Render(fmt.Sprintf("%s %s", m.getCheckBox(index), s))
		},
		renderFocused: func(s string, index int) string {
			if index == len(items)-1 {
				return xLargeLeftPadding.Copy().Inherit(focused).Render(s)
			}
			return focusTableItem(fmt.Sprintf("> %s %s", m.getCheckBox(index), s))
		},
	}

	m.list = createModelList(items, itemDelegate, 0)
	m.header = header
	m.itemMap = itemMap
	m.question = data.QuestionString

	// Create selected map and select defaults
	m.selected = make(map[int]item)
	defaultIndexes := getDefaultOptionIndexes(data)
	for _, defaultIndex := range defaultIndexes {
		m.selected[defaultIndex] = items[defaultIndex].(item)
	}
}

func (m *MultiSelectList) Init() tea.Cmd {
	return nil
}

func (m *MultiSelectList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.err = exitError
			return m, tea.Quit

		case tea.KeyEnter, tea.KeySpace:
			if m.list.Cursor() == len(m.list.Items())-1 {
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

func (m *MultiSelectList) View() string {
	b := strings.Builder{}
	if m.question != "" {
		b.WriteString(m.question + "\n\n")
	}

	if m.header != "" {
		b.WriteString(xLargeLeftPadding.Render(m.header) + "\n")
	}
	b.WriteString(m.list.View())
	return b.String()
}

func (m *MultiSelectList) GetSelectedValues() []string {
	values := make([]string, 0, len(m.selected))
	for _, value := range m.selected {
		values = append(values, m.itemMap[value])
	}
	return values
}

func (m *MultiSelectList) getCheckBox(checkBoxIndex int) string {
	checked := "[ ]"
	if _, ok := m.selected[checkBoxIndex]; ok {
		checked = "[x]"
	}
	return checked
}

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

func (m *MultiSelectList) getError() error { return m.err }
