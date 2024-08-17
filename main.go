package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var version string

func main() {
	if (os.Args[1] == "version") {
		fmt.Println(version)
		return
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	debug := os.Getenv("DEBUG")
	if debug == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal().Msg("GROQ_API_KEY is not set. Get one from https://console.groq.com/keys.")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		log.Fatal().Msg("Could not determine shell. Set SHELL environment variable.")
	}
	_shell := strings.Split(shell, "/")
	shell = _shell[len(_shell)-1]

	kernel := runtime.GOOS

	prompt := fmt.Sprintf(`Only reply with the single line command surrounded by three backticks. It must be able to be directly run in the target shell. Do not include any other text.
Make sure the command runs on %s shell on %s kernel.
The prompt: %s`, shell, kernel, strings.Join(os.Args[1:], " "))
	log.Debug().Msg(prompt)

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
