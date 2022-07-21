package questionModel

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var addButtonText = "ADD TAG"
var submitButtonText = "SUBMIT TAGS"

type keyValue struct {
	focusIndex          int
	submitButtonFocused bool
	inputs              []textinput.Model
	question            string
	tags                [][]string
	tagList             *SingleSelectList
	err                 error
}

func (kv *keyValue) InitializeModel(data *BubbleTeaData) {
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

	tags := strings.Split(data.DefaultOption, ",") //[tag1|val1, tag2|val2]
	for _, tag := range tags {
		pair := strings.Split(tag, "|") //[tag1, val1]
		kv.tags = append(kv.tags, []string{strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])})
	}

	tagList := &SingleSelectList{}
	tagList.InitializeModel(&BubbleTeaData{
		OptionData:    kv.tags,
		HeaderStrings: []string{"KEY", "VALUE"},
	})
	tagList.list.Select(-1)

	kv.tagList = tagList
	kv.question = data.QuestionString
}

func (kv *keyValue) Init() tea.Cmd {
	return textinput.Blink
}

func (kv *keyValue) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC:
			kv.err = errors.New("User has quit before finishing question!")
			return kv, tea.Quit

		case tea.KeyUp, tea.KeyDown, tea.KeyEnter, tea.KeyShiftTab, tea.KeyTab:
			msgType := msg.Type

			if msgType == tea.KeyEnter && kv.focusIndex == len(kv.inputs) {
				if !kv.submitButtonFocused {
					kv.addTag()
				} else {
					return kv, tea.Quit
				}
			}

			kv.cycleFocusIndex(msgType)

			if kv.focusIndex != len(kv.inputs) {
				kv.submitButtonFocused = false
			}

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
			if kv.focusIndex == len(kv.inputs) {
				kv.submitButtonFocused = !kv.submitButtonFocused
			}

		case tea.KeyBackspace:
			kv.deleteTag()
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

func (kv *keyValue) View() string {
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

func (kv *keyValue) addTag() {
	if kv.inputs[0].Value() == "" {
		kv.inputs[0].Placeholder = "Please Enter A Key!"
	}

	if kv.inputs[1].Value() == "" {
		kv.inputs[1].Placeholder = "Please Enter A Value!"
	}

	if kv.inputs[0].Value() != "" && kv.inputs[1].Value() != "" {
		kv.tags = append(kv.tags, []string{strings.TrimSpace(kv.inputs[0].Value()), strings.TrimSpace(kv.inputs[1].Value())})
		kv.tagList.InitializeModel(&BubbleTeaData{
			OptionData:    kv.tags,
			HeaderStrings: []string{"KEY", "VALUE"},
		})
		kv.inputs[0].Placeholder = "Key"
		kv.inputs[1].Placeholder = "Value"
		kv.inputs[0].SetValue("")
		kv.inputs[1].SetValue("")
	}
}

func (kv *keyValue) createButton(buttonText string, isFocused bool) string {
	button := blurred.Copy().Render(fmt.Sprintf("[ %s ]", buttonText))
	if kv.focusIndex == len(kv.inputs) && isFocused {
		button = fmt.Sprintf("[ %s ]", focused.Render(buttonText))
	}
	return button
}

func (kv *keyValue) cycleFocusIndex(msgType tea.KeyType) {
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

func (kv *keyValue) deleteTag() {
	cursor := kv.tagList.list.Cursor()
	if cursor >= 0 && len(kv.tagList.list.Items()) > 0 {
		kv.tags = append(kv.tags[:cursor], kv.tags[cursor+1:]...)
		kv.tagList.list.RemoveItem(cursor)
		kv.tagList.list.CursorUp()
		kv.tagList.list.SetHeight(kv.tagList.list.Height() - 1)
	}
}

func (kv *keyValue) focusInput(focusIndex int) tea.Cmd {
	kv.inputs[focusIndex].PromptStyle = smallLeftPadding.Copy().Inherit(focused)
	kv.inputs[focusIndex].TextStyle = focused
	return kv.inputs[focusIndex].Focus()
}

func (kv *keyValue) getError() error { return kv.err }

func (kv *keyValue) TagMapToString() string {
	builder := strings.Builder{}
	for index, tag := range kv.tags {
		builder.WriteString(fmt.Sprintf("%s|%s", tag[0], tag[1]))
		if index != len(kv.tags)-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}

func (kv *keyValue) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds = make([]tea.Cmd, len(kv.inputs))

	for i := range kv.inputs {
		kv.inputs[i], cmds[i] = kv.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}
