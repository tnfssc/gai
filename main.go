package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	args := os.Args
	args = args[1:]

	ctx := context.Background()
	stmt := strings.Join(args, " ")

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY is not set")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		log.Fatal("SHELL is not set")
	}

	prompt := fmt.Sprintf(`Only reply with the single line command surrounded by three backticks. It must be able to be directly run in the target shell. Do not include any other text.
Make sure the command runs on %s shell.
The prompt: %s`, shell, stmt)

	llm, err := openai.New(
		openai.WithModel("gemma2-9b-it"),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
		openai.WithToken(apiKey),
	)
	if err != nil {
		log.Fatal(err)
	}
	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		log.Fatal(err)
	}
	// remove the backticks
	completion = strings.Trim(completion, "`")
	completion = strings.Trim(completion, "`")
	completion = strings.TrimSpace(completion)
	fmt.Println(completion)

	if err := clipboard.WriteAll(string(completion)); err != nil {
		fmt.Println("Failed to copy to clipboard")
	}
}
