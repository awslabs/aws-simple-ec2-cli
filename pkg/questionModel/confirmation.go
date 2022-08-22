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
	"simple-ec2/pkg/cli"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

/*
Confirmation represents a question in which the user confirms a list of configurations. It
is comprised of two SingleSelectLists: The first is the list of configurations and the second
is the yes/no option to confirm the launch of the instance.
*/
type Confirmation struct {
	lists      []SingleSelectList
	choice     string // The chosen option
	focusIndex int    // The index of the cursor
	allowEdit  bool   // Whether the configurations list is selectable
	errorMsg   string // An error message to be presented if a config is selected which cant be reconfigured
	err        error  // An error caught during the question
}

// InitializeModel initializes the model based on the passed in question input
func (c *Confirmation) InitializeModel(input *QuestionInput) {
	configList := SingleSelectList{}
	configList.InitializeModel(&QuestionInput{
		HeaderStrings: []string{"Configuration", "Value"},
		QuestionString: "Please confirm if you would like to launch instance with following options" +
			"(Or select a configuration to repeat a question):",
		Rows:           input.Rows,
		IndexedOptions: input.IndexedOptions,
	})
	configList.list.Select(-1)

	yesNoList := SingleSelectList{}
	yesNoList.InitializeModel(&QuestionInput{
		IndexedOptions: yesNoOptions,
		DefaultOption:  cli.ResponseNo,
		Rows:           CreateSingleLineRows(yesNoData),
	})
	c.lists = append(c.lists, configList, yesNoList)
	c.focusIndex = 1
}

// Init defines an optional command that can be run when the question is asked.
func (c *Confirmation) Init() tea.Cmd {
	return nil
}

/*
Update is called when a message is received. Handles user input to traverse through lists and
select an answer.
*/
func (c *Confirmation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			c.err = exitError
			return c, tea.Quit

		case tea.KeyUp:
			// Decrease the cursor index if there are more elements to move up to, and if they're allowed
			// to be focused on.
			if c.focusIndex > -len(c.lists[0].list.Items()) && (c.allowEdit || c.focusIndex > 0) {
				c.focusIndex--
			}
			// Focus the config list and unfocus the yes/no list
			if c.focusIndex == -1 {
				c.lists[0].list.Select(len(c.lists[0].list.Items()))
				c.lists[1].list.Select(c.focusIndex)
			}

		case tea.KeyDown:
			// Increase the cursor index if there are more elements to move down to
			if c.focusIndex < len(c.lists[1].list.Items())-1 {
				c.focusIndex++
			}
			// Focus the yes/no list and unfocus the config list
			if c.focusIndex == 0 {
				c.lists[0].list.Select(-1)
				c.lists[1].list.Select(c.focusIndex)
				return c, nil
			}

		case tea.KeyEnter:
			c.errorMsg = ""
			// Select an item from the appropriate list
			if c.focusIndex < 0 {
				c.lists[0].selectItem()
				c.choice = c.lists[0].GetChoice()
				// Set an error message if there is no choice associated with the option
				if c.choice == "" {
					c.errorMsg = "This configuration can't be modified!"
					return c, nil
				}
				return c, tea.Quit
			} else {
				c.lists[1].selectItem()
				c.choice = c.lists[1].GetChoice()
				return c, tea.Quit
			}
		}

	case error:
		c.err = msg
		return c, tea.Quit
	}

	// Update the appropriate list
	if c.focusIndex < 0 && c.allowEdit {
		c.lists[0].Update(msg)
	} else {
		c.lists[1].Update(msg)
	}
	return c, nil
}

// View renders the view for the question. The view is rendered after every update
func (c *Confirmation) View() string {
	b := strings.Builder{}
	if c.errorMsg != "" {
		b.WriteString(errorStyle.Render(c.errorMsg) + "\n")
	}
	b.WriteString(c.lists[0].View())
	b.WriteRune('\n')
	b.WriteString(c.lists[1].View())
	return b.String()
}

// GetChoice gets the selected choice
func (c *Confirmation) GetChoice() string { return c.choice }

// getError gets the error from the question if one arose
func (c *Confirmation) GetError() error { return c.err }

// SetAllowEdit sets whether the configuration list can be selected or not
func (c *Confirmation) SetAllowEdit(allowEdit bool) {
	c.allowEdit = allowEdit
}
