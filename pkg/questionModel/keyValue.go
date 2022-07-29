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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Text values for the "Add" and "Submit" buttons
const (
	addButtonText    = "ADD TAG"
	submitButtonText = "SUBMIT TAGS"
)

// Headers for the list of created tags
var tagHeaders = []string{"KEY", "VALUE"}

// KeyValue represents a question where the user is asked for key value pairs
type KeyValue struct {
	focusIndex          int
	submitButtonFocused bool
	inputs              []textinput.Model
	question            string
	tags                [][]string
	tagList             *SingleSelectList
	err                 error
}

// InitializeModel initializes the model based on the passed in question input
func (kv *KeyValue) InitializeModel(input *QuestionInput) {
	kv.inputs = make([]textinput.Model, 2)

	var t textinput.Model
	for i := range kv.inputs {
		t = textinput.New()
		t.CursorStyle = focused
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = "Key"
			t.Focus()
			t.PromptStyle = smallLeftPadding.Copy().Inherit(focused)
			t.TextStyle = focused
		case 1:
			t.Placeholder = "Value"
			t.PromptStyle = smallLeftPadding
		}

		kv.inputs[i] = t
	}

	// Populates the kv.tags with default tags
	tags := strings.Split(input.DefaultOption, ",") //[tag1|val1, tag2|val2]
	for _, tag := range tags {
		pair := strings.Split(tag, "|") //[tag1, val1]
		if len(pair) == 2 {
			kv.tags = append(kv.tags, []string{strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])})
		}
	}

	// Initializes the created tag list
	tagList := &SingleSelectList{}
	tagList.InitializeModel(&QuestionInput{
		OptionData:    kv.tags,
		HeaderStrings: tagHeaders,
	})
	tagList.list.Select(-1)

	kv.tagList = tagList
	kv.question = input.QuestionString
}

// Init defines an optional command that can be run when the question is asked.
func (kv *KeyValue) Init() tea.Cmd {
	return textinput.Blink
}

/*
Update is called when a message is received. Handles user input to add new tags, delete old tags,
move the cursor, and submit the tags
*/
func (kv *KeyValue) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC:
			kv.err = exitError
			return kv, tea.Quit

		case tea.KeyUp, tea.KeyDown, tea.KeyEnter, tea.KeyShiftTab, tea.KeyTab:
			msgType := msg.Type

			// If a button is pressed, identify which one is pressed and act accordingly
			if msgType == tea.KeyEnter && kv.areButtonsFocused() {
				if kv.submitButtonFocused {
					return kv, tea.Quit
				} else {
					kv.addTag()
				}
			}

			kv.cycleFocusIndex(msgType)

			// Reset focus to the "Add Tag" button when navigating away from the buttons
			if !kv.areButtonsFocused() {
				kv.submitButtonFocused = false
			}

			// Focus or unfocus text inputs
			cmds := make([]tea.Cmd, len(kv.inputs))
			for i := 0; i <= len(kv.inputs)-1; i++ {
				if i == kv.focusIndex {
					cmds[i] = kv.focusInput(i)
					continue
				}
				kv.inputs[i].Blur()
				kv.inputs[i].PromptStyle = smallLeftPadding
				kv.inputs[i].TextStyle = noStyle
			}

			return kv, tea.Batch(cmds...)

		case tea.KeyRight, tea.KeyLeft:
			if kv.areButtonsFocused() {
				kv.submitButtonFocused = !kv.submitButtonFocused
			}

		case tea.KeyBackspace:
			kv.deleteTag()
			// If there are no more tags then set the focus back to the first text input
			if len(kv.tagList.list.Items()) == 0 {
				kv.focusIndex = 0
				return kv, kv.focusInput(kv.focusIndex)
			}
		}

	case error:
		kv.err = msg
		return kv, tea.Quit
	}

	cmd := kv.updateInputs(msg)
	return kv, cmd
}

// View renders the view for the question. The view is rendered after every update
func (kv *KeyValue) View() string {
	var b strings.Builder
	if kv.question != "" {
		b.WriteString(kv.question + "\n")
	}

	if len(kv.tagList.list.Items()) > 0 {
		b.WriteString("\n" + kv.tagList.View() + "\n")
	} else {
		b.WriteRune('\n')
	}

	for i := range kv.inputs {
		b.WriteString(kv.inputs[i].View())
		if i < len(kv.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	addButton := kv.createButton(addButtonText, !kv.submitButtonFocused)
	submitButton := kv.createButton(submitButtonText, kv.submitButtonFocused)
	b.WriteString(fmt.Sprintf(smallLeftPadding.Render("\n\n%s  %s\n"), addButton, submitButton))

	return b.String()
}

// addTag creates and adds a tag to the list of tags
func (kv *KeyValue) addTag() {
	if kv.inputs[0].Value() == "" {
		kv.inputs[0].Placeholder = "Please Enter A Key!"
	}

	if kv.inputs[1].Value() == "" {
		kv.inputs[1].Placeholder = "Please Enter A Value!"
	}

	if kv.inputs[0].Value() != "" && kv.inputs[1].Value() != "" {
		kv.tags = append(kv.tags, []string{strings.TrimSpace(kv.inputs[0].Value()), strings.TrimSpace(kv.inputs[1].Value())})
		kv.tagList.InitializeModel(&QuestionInput{
			OptionData:    kv.tags,
			HeaderStrings: tagHeaders,
		})
		kv.inputs[0].Placeholder = "Key"
		kv.inputs[1].Placeholder = "Value"
		kv.inputs[0].SetValue("")
		kv.inputs[1].SetValue("")
	}
}

// areButtonsFocused determines if one of the buttons are focused
func (kv *KeyValue) areButtonsFocused() bool { return kv.focusIndex == len(kv.inputs) }

// createButton creates a button with given text and determines if that button is focused or not
func (kv *KeyValue) createButton(buttonText string, isFocused bool) string {
	button := blurred.Copy().Render(fmt.Sprintf("[ %s ]", buttonText))
	if kv.areButtonsFocused() && isFocused {
		button = fmt.Sprintf("[ %s ]", focused.Render(buttonText))
	}
	return button
}

// cycleFocusIndex cycles through the focus index based on the key that the user enters.
// This determines how the cursor focuses on rows in the tag list, text entries, and the button row
func (kv *KeyValue) cycleFocusIndex(msgType tea.KeyType) {
	if msgType == tea.KeyUp || msgType == tea.KeyShiftTab {
		kv.focusIndex--
	} else {
		kv.focusIndex++
	}

	if kv.focusIndex >= 0 {
		kv.tagList.list.Select(-1)
		if kv.focusIndex > len(kv.inputs) {
			kv.focusIndex = 0
		}
	} else {
		if len(kv.tagList.list.Items()) == 0 {
			kv.focusIndex = 0
		}
		if kv.focusIndex < -len(kv.tagList.list.Items()) {
			kv.focusIndex = -len(kv.tagList.list.Items())
		}
		kv.tagList.list.Select(len(kv.tagList.list.Items()) + kv.focusIndex)
	}
}

// deleteTag deletes a tag if cursor is focused on a tag in the tag list
func (kv *KeyValue) deleteTag() {
	cursor := kv.tagList.list.Cursor()
	if cursor >= 0 && len(kv.tagList.list.Items()) > 0 {
		kv.tags = append(kv.tags[:cursor], kv.tags[cursor+1:]...)
		kv.tagList.list.RemoveItem(cursor)
		kv.tagList.list.CursorUp()
		kv.tagList.list.SetHeight(kv.tagList.list.Height() - 1)
	}
}

// focusInput focuses one of the text inputs based on the given index
func (kv *KeyValue) focusInput(focusIndex int) tea.Cmd {
	kv.inputs[focusIndex].PromptStyle = smallLeftPadding.Copy().Inherit(focused)
	kv.inputs[focusIndex].TextStyle = focused
	return kv.inputs[focusIndex].Focus()
}

// getError gets the error from the question if one arose
func (kv *KeyValue) getError() error { return kv.err }

// TagsToString returns a string value of the created tags
func (kv *KeyValue) TagsToString() string {
	builder := strings.Builder{}
	for index, tag := range kv.tags {
		builder.WriteString(fmt.Sprintf("%s|%s", tag[0], tag[1]))
		if index != len(kv.tags)-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}

// updateInputs updates the text inputs based on user entry
func (kv *KeyValue) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds = make([]tea.Cmd, len(kv.inputs))

	for i := range kv.inputs {
		kv.inputs[i], cmds[i] = kv.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}
