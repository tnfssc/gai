package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	debug := os.Getenv("DEBUG")
	if debug == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal().Msg("GROQ_API_KEY is not set")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		log.Fatal().Msg("SHELL is not set")
	}

	prompt := fmt.Sprintf(`Only reply with the single line command surrounded by three backticks. It must be able to be directly run in the target shell. Do not include any other text.
Make sure the command runs on %s shell.
The prompt: %s`, shell, strings.Join(os.Args[1:], " "))

	llm, err := openai.New(
		openai.WithModel("gemma2-9b-it"),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
		openai.WithToken(apiKey),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create llm")
	}
	completion, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate completion")
	}
	fmt.Println(strings.TrimSpace(strings.Trim(completion, "`")))
	if err := clipboard.WriteAll(string(completion)); err != nil {
		log.Debug().Err(err).Msg("Failed to copy to clipboard")
	}
}
