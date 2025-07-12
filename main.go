package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var version string

type CacheEntry struct {
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type Config struct {
	GroqAPIKey       string `json:"groq_api_key"`
	OpenRouterAPIKey string `json:"openrouter_api_key"`
	DefaultProvider  string `json:"default_provider"`
	GroqModel        string `json:"groq_model"`
	OpenRouterModel  string `json:"openrouter_model"`
}

func getConfigDir() string {
	var configDir string
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(os.Getenv("APPDATA"), "gai")
	} else {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "gai")
	}
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0700)
	}
	return configDir
}

func loadConfig() (Config, error) {
	configDir := getConfigDir()
	configFile := filepath.Join(configDir, "gai.config.json")
	var config Config
	f, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func saveConfig(config Config) error {
	configDir := getConfigDir()
	configFile := filepath.Join(configDir, "gai.config.json")
	f, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(config)
}

func getCacheDir() string {
	cacheDir := os.TempDir()
	return filepath.Join(cacheDir, "gai-cache")
}

func hashPrompt(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:])
}

func loadCache(cacheKey string) (string, bool) {
	cacheDir := getCacheDir()
	cacheFile := filepath.Join(cacheDir, cacheKey+".json")

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return "", false
	}

	file, err := os.Open(cacheFile)
	if err != nil {
		return "", false
	}
	defer file.Close()

	var entry CacheEntry
	if err := json.NewDecoder(file).Decode(&entry); err != nil {
		return "", false
	}

	if time.Now().After(entry.Timestamp.Add(2400 * time.Hour)) {
		os.Remove(cacheFile)
		return "", false
	}

	return entry.Value, true
}

func saveCache(cacheKey string, value string) error {
	cacheDir := getCacheDir()
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return err
		}
	}

	cacheFile := filepath.Join(cacheDir, cacheKey+".json")
	file, err := os.Create(cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	entry := CacheEntry{
		Value:     value,
		Timestamp: time.Now(),
	}
	return json.NewEncoder(file).Encode(entry)
}

func writeToClipboard(text string) {
	if err := clipboard.WriteAll(text); err == nil {
		fmt.Println()
		fmt.Println(string("\033[32m"), "Copied \uf05d", string("\033[0m"))
	}
}

func generateCommand(prompt, provider, apiKey, model string) (string, error) {
	var baseURL string

	switch provider {
	case "openrouter":
		baseURL = "https://openrouter.ai/api/v1"
	default: // groq
		baseURL = "https://api.groq.com/openai/v1"
	}

	llm, err := openai.New(
		openai.WithModel(model),
		openai.WithBaseURL(baseURL),
		openai.WithToken(apiKey),
	)
	if err != nil {
		return "", err
	}

	completionRaw, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.Trim(completionRaw, "`")), nil
}

func getStdinText() string {
	stdinText := ""
	fi, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			stdinText += (scanner.Text() + "\n")
		}
		if err := scanner.Err(); err != nil {
			return ""
		}
		stdinText = strings.TrimSpace(stdinText)
		if len(stdinText) > 0 {
			stdinText = fmt.Sprintf(`
<Additional Context Start>
%s
<Additional Context End>
`, stdinText)
		}
	}
	return stdinText
}

func getShell() string {
	shellEnv := os.Getenv("SHELL")
	if shellEnv == "" {
		fmt.Println("Could not determine shell. Set SHELL environment variable.")
		os.Exit(1)
	}
	parts := strings.Split(shellEnv, "/")
	return parts[len(parts)-1]
}

func testAPIKey(provider, apiKey, model string) error {
	var baseURL string

	switch provider {
	case "openrouter":
		baseURL = "https://openrouter.ai/api/v1"
	default: // groq
		baseURL = "https://api.groq.com/openai/v1"
	}

	llm, err := openai.New(
		openai.WithModel(model),
		openai.WithBaseURL(baseURL),
		openai.WithToken(apiKey),
	)
	if err != nil {
		return err
	}

	_, err = llms.GenerateFromSinglePrompt(context.Background(), llm, "Say hello")
	return err
}

func promptForInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func ensureDefaultProvider(config *Config) {
	if config.DefaultProvider == "" {
		fmt.Println("No default provider configured.")
		fmt.Println("Available providers: groq, openrouter")
		for {
			provider := promptForInput("Please enter your preferred default provider (groq/openrouter): ")
			if provider == "groq" || provider == "openrouter" {
				config.DefaultProvider = provider
				break
			}
			fmt.Println("Invalid provider. Please choose 'groq' or 'openrouter'.")
		}
	}
}

func ensureModelForProvider(config *Config, provider string) {
	switch provider {
	case "groq":
		if config.GroqModel == "" {
			fmt.Println("No Groq model configured.")
			fmt.Println("Popular Groq models: llama-3.3-70b-specdec, meta-llama/llama-4-maverick-17b-128e-instruct, llama-3.1-70b-versatile")
			config.GroqModel = promptForInput("Please enter the Groq model name: ")
		}
	case "openrouter":
		if config.OpenRouterModel == "" {
			fmt.Println("No OpenRouter model configured.")
			fmt.Println("Popular OpenRouter models: deepseek/deepseek-chat:free, anthropic/claude-3.5-sonnet, openai/gpt-4o")
			config.OpenRouterModel = promptForInput("Please enter the OpenRouter model name: ")
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [--provider PROVIDER] [PROMPT]")
		os.Exit(1)
	}

	if os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	config, _ := loadConfig()

	// Ensure default provider is configured
	ensureDefaultProvider(&config)

	provider := config.DefaultProvider

	// Setup flags after handling version check
	flag.StringVar(&provider, "provider", config.DefaultProvider, "llm provider: groq or openrouter")
	flag.Parse()

	// Remaining args after flag parsing are the prompt tokens
	promptArgs := flag.Args()
	if len(promptArgs) == 0 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [--provider PROVIDER] [PROMPT]")
		os.Exit(1)
	}

	// Ensure model is configured for the selected provider
	ensureModelForProvider(&config, provider)

	// Save configuration if it was updated
	if err := saveConfig(config); err != nil {
		fmt.Println("Failed to save config to file.")
	}

	var apiKey string

	var model string
	switch provider {
	case "openrouter":
		model = config.OpenRouterModel
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			apiKey = config.OpenRouterAPIKey
		}
		if apiKey == "" {
			fmt.Print("OPENROUTER_API_KEY is not set in env or config. Get one from https://openrouter.ai. Please enter your OPENROUTER_API_KEY: ")
			reader := bufio.NewReader(os.Stdin)
			inputAPIKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(inputAPIKey)

			if err := testAPIKey(provider, apiKey, model); err != nil {
				fmt.Println("OPENROUTER_API_KEY is invalid:", err)
				fmt.Println("Please check your API key or get a new one from https://openrouter.ai")
				os.Exit(1)
			}

			config.OpenRouterAPIKey = apiKey
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save config to file.")
			}
		}
		if apiKey == "" {
			fmt.Println("OPENROUTER_API_KEY is required. Get one from https://openrouter.ai")
			os.Exit(1)
		}
	default: // groq
		model = config.GroqModel
		apiKey = os.Getenv("GROQ_API_KEY")
		if apiKey == "" {
			apiKey = config.GroqAPIKey
		}
		if apiKey == "" {
			fmt.Print("GROQ_API_KEY is not set in env or config. You can get one from https://console.groq.com/keys. Please enter your GROQ_API_KEY: ")
			reader := bufio.NewReader(os.Stdin)
			inputAPIKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(inputAPIKey)

			if err := testAPIKey(provider, apiKey, model); err != nil {
				fmt.Println("GROQ_API_KEY is invalid:", err)
				fmt.Println("Please check your API key or get a new one from https://console.groq.com/keys")
				os.Exit(1)
			}

			config.GroqAPIKey = apiKey
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save config to file.")
			}
		}
		if apiKey == "" {
			fmt.Println("GROQ_API_KEY is required. Get one from https://console.groq.com/keys")
			os.Exit(1)
		}
	}

	shell := getShell()
	kernel := runtime.GOOS
	stdinText := getStdinText()

	prompt := fmt.Sprintf(`
<Instructions Start>
Only reply with the single line command surrounded by three backticks.
It must be able to be directly run in the target shell.
Do not include any other text.
Make sure the command runs on %s shell on %s kernel.
<Instructions End>

%s

<Prompt Start>
%s
<Prompt End>
`, shell, kernel, stdinText, strings.Join(promptArgs, " "))

	promptHash := hashPrompt(prompt)
	if cachedResponse, ok := loadCache(promptHash); ok {
		fmt.Println()
		fmt.Println("", cachedResponse)
		writeToClipboard(cachedResponse)
		return
	}

	completion, err := generateCommand(prompt, provider, apiKey, model)
	if err != nil {
		fmt.Println("Failed to generate command", err)
		os.Exit(1)
	}

	if err := saveCache(promptHash, completion); err != nil {
		fmt.Println("Failed to save cache")
	}

	fmt.Println()
	fmt.Println("", completion)
	writeToClipboard(completion)
}
