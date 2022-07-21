package questionModel

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type MultiSelectList struct {
	list     list.Model
	selected map[int]item // which to-do items are selected
	itemMap  map[item]string
	header   string
	question string
	err      error
}

func (m *MultiSelectList) InitializeModel(data *BubbleTeaData) {
	header, items, itemMap := createItems(data)
	items = append(items, item("\n[ Submit ]"))

	itemDelegate := itemDelegate{
		renderUnselected: func(s string, index int) string {
			if index == len(items)-1 {
				return xLargeLeftPadding.Copy().Inherit(blurred).Render(s)
			}
			return mediumLeftPadding.Render(fmt.Sprintf("%s %s", m.getCheckBox(index), s))
		},
		renderSelected: func(s string, index int) string {
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
			m.err = errors.New("User has quit before finishing question!")
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
