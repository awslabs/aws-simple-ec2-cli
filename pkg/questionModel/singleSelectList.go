package questionModel

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type SingleSelectList struct {
	list     list.Model
	choice   string
	itemMap  map[item]string
	header   string
	question string
	err      error
}

func (s *SingleSelectList) InitializeModel(data *BubbleTeaData) {
	header, items, itemMap := createItems(data)

	itemDelegate := itemDelegate{
		renderUnselected: func(s string, index int) string {
			return mediumLeftPadding.Render(s)
		},
		renderSelected: func(s string, index int) string {
			return focusTableItem("> " + s)
		},
	}

	defaultOptionIndex := getDefaultOptionIndex(data)
	if defaultOptionIndex == -1 {
		defaultOptionIndex = 0
	}

	s.list = createModelList(items, itemDelegate, defaultOptionIndex)
	s.header = header
	s.itemMap = itemMap
	s.question = data.QuestionString
}

func (s *SingleSelectList) Init() tea.Cmd {
	return nil
}

func (s *SingleSelectList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetWidth(msg.Width)
		return s, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			s.err = errors.New("User has quit before finishing question!")
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

func (s *SingleSelectList) GetChoice() string { return s.choice }

func (s *SingleSelectList) getError() error { return s.err }

func (s *SingleSelectList) selectItem() {
	i, ok := s.list.SelectedItem().(item)
	if ok {
		s.choice = s.itemMap[i]
	}
}
