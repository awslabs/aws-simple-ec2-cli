package testhelper

import (
	"simple-ec2/pkg/questionModel"

	tea "github.com/charmbracelet/bubbletea"
)

type MockedQMHelperSvc struct {
	UserInputs []tea.Msg
}

func (m *MockedQMHelperSvc) AskQuestion(model questionModel.QuestionModel, questionInput *questionModel.QuestionInput) error {
	var err error
	model.InitializeModel(questionInput)
	for _, input := range m.UserInputs {
		model.Update(input)
		if model.GetError() != nil {
			err = model.GetError()
			return err
		}
	}
	return err
}
