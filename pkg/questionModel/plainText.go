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

type PlainText struct {
	textInput         textinput.Model
	question          string
	validFunctions    []CheckInput
	EC2Helper         *ec2helper.EC2Helper
	invalidMsg        string
	displayInvalidMsg bool
	err               error
}

func (pt *PlainText) InitializeModel(data *BubbleTeaData) {
	ti := textinput.New()
	ti.Placeholder = data.DefaultOption
	ti.Focus()

	pt.textInput = ti
	pt.question = data.QuestionString
	pt.validFunctions = data.Fns
	pt.EC2Helper = data.EC2Helper
}

func (pt *PlainText) Init() tea.Cmd {
	return textinput.Blink
}

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
			if pt.isValidInput(pt.textInput.Value()) {
				pt.displayInvalidMsg = false
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

func (pt *PlainText) getError() error { return pt.err }

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

func (pt *PlainText) GetTextAnswer() string { return pt.textInput.Value() }
