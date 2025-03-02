package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var version string

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [PROMPT]")
		os.Exit(1)
	}

	if os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("GROQ_API_KEY is not set. Get one from https://console.groq.com/keys")
		os.Exit(1)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		fmt.Println("Could not determine shell. Set SHELL environment variable.")
		os.Exit(1)
	}
	_shell := strings.Split(shell, "/")
	shell = _shell[len(_shell)-1]

	kernel := runtime.GOOS

	prompt := fmt.Sprintf(`Only reply with the single line command surrounded by three backticks. It must be able to be directly run in the target shell. Do not include any other text.
Make sure the command runs on %s shell on %s kernel.
The prompt: %s`, shell, kernel, strings.Join(os.Args[1:], " "))

	llm, _ := openai.New(
		openai.WithModel("llama-3.3-70b-specdec"),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
		openai.WithToken(apiKey),
	)

	completion, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
	if err != nil {
		fmt.Println("Failed to generate command")
		os.Exit(1)
	}

	completion = fmt.Sprint(strings.TrimSpace(strings.Trim(completion, "`")))
	fmt.Println()
	fmt.Println("", completion)

	if err := clipboard.WriteAll(completion); err == nil {
		fmt.Println()
		fmt.Println(string("\033[32m"), "Copied \uf05d")
	}
}
