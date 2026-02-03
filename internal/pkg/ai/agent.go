package ai

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
)

type CommandAgent struct {
	genkit  *genkit.Genkit
	context context.Context
}

func NewCommandAgent(
	provider api.Plugin,
	context context.Context,
) *CommandAgent {
	g := genkit.Init(
		context,
		genkit.WithPlugins(
			provider,
		),
		genkit.WithDefaultModel(fmt.Sprint(PROVIDER_NAME, "/", MODEL_NAME)),
	)

	return &CommandAgent{
		genkit:  g,
		context: context,
	}
}

func (a *CommandAgent) Generate(prompt string) (string, error) {
	res, err := genkit.Generate(
		a.context,
		a.genkit,
		ai.WithMessages(INITIAL_MESSAGES...),
		ai.WithPrompt(prompt),
	)

	if err != nil {
		return "", err
	}

	return res.Text(), nil
}

var INITIAL_MESSAGES = []*ai.Message{
	{
		Role: ai.RoleSystem,
		Content: []*ai.Part{
			{
				Text: `You are a command generating assistant. The user will give you a query you will generate a command to execute.` +
					`Your output should either by ONLY a command, or if you cannot produce the command then respond with "IDK"` +
					"IMPORTANT: when outputting a command, DO NOT include any additional text or formatting like quotes or backticks" +
					`The user is in a ` + getShell() + ` shell`,
			},
		},
	},
}
