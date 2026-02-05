package ai

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
)

type CommandAgent struct {
	genkit   *genkit.Genkit
	context  context.Context
	messages []*ai.Message
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
		messages: []*ai.Message{
			{
				Role: ai.RoleSystem,
				Content: []*ai.Part{
					{
						Text: getCommandGenerationSystemPrompt(),
					},
				},
			},
		},
	}
}

func (a *CommandAgent) Generate(prompt string) (string, error) {
	a.messages = append(a.messages, &ai.Message{
		Role: ai.RoleUser,
		Content: []*ai.Part{
			{
				Text: prompt,
			},
		},
	})

	res, err := genkit.Generate(
		a.context,
		a.genkit,
		ai.WithMessages(a.messages...),
	)

	if err != nil {
		return "", err
	}

	a.messages = append(a.messages, res.Message)

	return res.Text(), nil
}

func (a *CommandAgent) Explain(prompt string, command string) (string, error) {
	res, err := genkit.Generate(
		a.context,
		a.genkit,
		ai.WithSystem(getExplainSystemPrompt()),
		ai.WithPrompt(
			fmt.Sprint(
				"The user asked for a command to do the following: ", prompt, "\n",
				"The command generated was: ", command, "\n",
				"Explain what this command does.",
			),
		),
	)

	if err != nil {
		return "", err
	}

	return res.Text(), nil
}

func getCommandGenerationSystemPrompt() string {
	return `You are a command generating assistant. The user will give you a query you will generate a command to execute.
Your output should either by ONLY a command, or if you cannot produce the command then respond with "IDK"

If the command requires a specific directory and the user has not provided one, use the current directory (".").

IMPORTANT: when outputting a command, DO NOT include any additional text or formatting like quotes or backticks` +
		fmt.Sprint(`The user is in a`, getShell(), `shell`)
}

func getExplainSystemPrompt() string {
	return `You are a command explanation assistant, operating in a MacOS terminal.
Based on the provided command, explain what it does.
The user is a technical person (Software Engineer), so keep your explanation concise and to the point.

IMPORTANT: your output should only include a single explanation, no command or additional text.
Do not include any backticks or other special characters either` +
		fmt.Sprint(`The user is in a`, getShell(), `shell`)
}
