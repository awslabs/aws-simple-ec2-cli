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
	"simple-ec2/pkg/ec2helper"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// PlainText represents a question with a text input
type PlainText struct {
	textInput         textinput.Model      // The text input
	question          string               // The question being asked
	validFunctions    []CheckInput         // List of functions to validate the input
	EC2Helper         *ec2helper.EC2Helper // EC2Helper to provide validation methods for text inputs
	invalidMsg        string               // Message to display if input is invalid
	displayInvalidMsg bool                 // If the invalid message should be displayed or not
	err               error                // An error caught during the question

}

// InitializeModel initializes the model based on the passed in question input
func (pt *PlainText) InitializeModel(input *QuestionInput) {
	ti := textinput.New()
	ti.Placeholder = input.DefaultOption
	ti.Focus()

	pt.textInput = ti
	pt.question = input.QuestionString
	pt.validFunctions = input.Fns
	pt.EC2Helper = input.EC2Helper
}

// Init defines an optional command that can be run when the question is asked.
func (pt *PlainText) Init() tea.Cmd {
	return textinput.Blink
}

/*
Update is called when a message is received. Handles user input to enter text input and
inform user if the input is invalid
*/
func (pt *PlainText) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			pt.err = exitError
			return pt, tea.Quit

		case tea.KeyEnter:
			if pt.textInput.Value() == "" {
				pt.textInput.SetValue(pt.textInput.Placeholder)
			}
			// If input is valid quit. If invalid display error msg and reset input text
			if pt.isValidInput(pt.textInput.Value()) {
				pt.displayInvalidMsg = false
				pt.textInput.SetCursorMode(textinput.CursorHide)
				return pt, tea.Quit
			} else {
				pt.invalidMsg = pt.textInput.Value()
				pt.displayInvalidMsg = true
				pt.textInput.SetValue("")
				return pt, nil
			}
		}

	case error:
		pt.err = msg
		return pt, tea.Quit
	}

	pt.textInput, cmd = pt.textInput.Update(msg)
	return pt, cmd
}

// View renders the view for the question. The view is rendered after every update
func (pt *PlainText) View() string {
	b := strings.Builder{}
	if pt.question != "" {
		b.WriteString(pt.question + "\n\n")
	}
	if pt.displayInvalidMsg {
		b.WriteString(smallLeftPadding.Copy().Inherit(errorStyle).Render(fmt.Sprintf("%s is an invalid answer. Enter a valid answer.", pt.invalidMsg)) + "\n")
	}
	b.WriteString(smallLeftPadding.Render(pt.textInput.View()) + "\n")
	return b.String()
}

// getError gets the error from the question if one arose
func (pt *PlainText) getError() error { return pt.err }

// isValidInput determines whether the answer is valid based on PlainText's validFunctions attribute
func (pt *PlainText) isValidInput(answer string) bool {
	if pt.EC2Helper != nil && pt.validFunctions != nil {
		for _, fn := range pt.validFunctions {
			if fn(pt.EC2Helper, answer) {
				return true
			}
		}
		return false
	}
	return true
}

// GetTextAnswer gets the answer from the text entry
func (pt *PlainText) GetTextAnswer() string { return pt.textInput.Value() }
