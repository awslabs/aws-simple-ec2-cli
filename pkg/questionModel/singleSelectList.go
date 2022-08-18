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
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

/*
SingleSelectList represents a question with a list of options from which a single option is chosen as the answer.
Options may be presented in a table based on initialized input.
*/
type SingleSelectList struct {
	list     list.Model      // The list of options
	choice   string          // The chosen option
	itemMap  map[item]string // Maps the item chosen to the answer value
	header   string          // The header for the item list table
	question string          // The question being asked
	err      error           // An error caught during the question
}

// InitializeModel initializes the model based on the passed in question input
func (s *SingleSelectList) InitializeModel(input *QuestionInput) {
	header, items, itemMap := createItems(input)

	// Define how list items are rendered in their focused and unfocused states
	itemDelegate := itemDelegate{
		renderUnfocused: func(s string, index int) string {
			return mediumLeftPadding.Render(s)
		},
		renderFocused: func(s string, index int) string {
			return styleTableItemRows("> "+s, mediumLeftPadding, focused, smallLeftPadding.Copy().Inherit(focused))
		},
	}

	defaultOptionIndex := getDefaultOptionIndex(input)
	if defaultOptionIndex == -1 {
		defaultOptionIndex = 0
	}

	s.list = createModelList(items, itemDelegate, defaultOptionIndex)
	s.header = header
	s.itemMap = itemMap
	s.question = input.QuestionString
}

// Init defines an optional command that can be run when the question is asked.
func (s *SingleSelectList) Init() tea.Cmd {
	return nil
}

/*
Update is called when a message is received. Handles user input to traverse through list and
select an answer.
*/
func (s *SingleSelectList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			s.err = exitError
			return s, tea.Quit

		case tea.KeyEnter:
			s.selectItem()
			return s, tea.Quit
		}

	case error:
		s.err = msg
		return s, tea.Quit
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

// View renders the view for the question. The view is rendered after every update
func (s *SingleSelectList) View() string {
	b := strings.Builder{}
	if s.question != "" {
		b.WriteString(s.question + "\n\n")
	}

	if s.header != "" {
		b.WriteString(mediumLeftPadding.Render(s.header) + "\n")
	}
	b.WriteString(s.list.View())
	return b.String()
}

// GetChoice gets the selected choice
func (s *SingleSelectList) GetChoice() string { return s.choice }

// getError gets the error from the question if one arose
func (s *SingleSelectList) GetError() error { return s.err }

// selectItem selects the focused item in the list
func (s *SingleSelectList) selectItem() {
	i, ok := s.list.SelectedItem().(item)
	if ok {
		s.choice = s.itemMap[i]
	}
}

// PrintTable prints the selection table
func (s *SingleSelectList) PrintTable() string {
	s.list.Select(-1)
	return s.View()
}
