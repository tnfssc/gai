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
	GroqAPIKey      string `json:"groq_api_key"`
	OpenRouterAPIKey string `json:"openrouter_api_key"`
	DefaultProvider string `json:"default_provider,omitempty"`
	DefaultModel    string `json:"default_model,omitempty"`
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
		if model == "" {
			model = "deepseek/deepseek-chat:free"
		}
	default: // groq
		baseURL = "https://api.groq.com/openai/v1"
		if model == "" {
			model = "meta-llama/llama-4-maverick-17b-128e-instruct"
		}
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

func testAPIKey(provider, apiKey string) error {
	var (
		model   string
		baseURL string
	)

	switch provider {
	case "openrouter":
		model = "deepseek/deepseek-chat:free"
		baseURL = "https://openrouter.ai/api/v1"
	default: // groq
		model = "meta-llama/llama-4-maverick-17b-128e-instruct"
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

func getModelChoice(provider string) string {
	reader := bufio.NewReader(os.Stdin)
	
	switch provider {
	case "openrouter":
		fmt.Println("\nAvailable OpenRouter models:")
		fmt.Println("1. deepseek/deepseek-chat:free (default)")
		fmt.Println("2. google/gemini-2.0-flash-exp:free")
		fmt.Println("3. meta-llama/llama-3.2-3b-instruct:free")
		fmt.Println("4. Custom model (enter full model name)")
		fmt.Print("\nSelect model (1-4) or press Enter for default: ")
		
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		
		switch choice {
		case "", "1":
			return "deepseek/deepseek-chat:free"
		case "2":
			return "google/gemini-2.0-flash-exp:free"
		case "3":
			return "meta-llama/llama-3.2-3b-instruct:free"
		case "4":
			fmt.Print("Enter custom model name: ")
			model, _ := reader.ReadString('\n')
			return strings.TrimSpace(model)
		default:
			return "deepseek/deepseek-chat:free"
		}
		
	default: // groq
		fmt.Println("\nAvailable Groq models:")
		fmt.Println("1. meta-llama/llama-4-maverick-17b-128e-instruct (default)")
		fmt.Println("2. llama-3.3-70b-versatile")
		fmt.Println("3. llama-3.1-8b-instant")
		fmt.Println("4. mixtral-8x7b-32768")
		fmt.Println("5. gemma2-9b-it")
		fmt.Println("6. Custom model (enter full model name)")
		fmt.Print("\nSelect model (1-6) or press Enter for default: ")
		
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		
		switch choice {
		case "", "1":
			return "meta-llama/llama-4-maverick-17b-128e-instruct"
		case "2":
			return "llama-3.3-70b-versatile"
		case "3":
			return "llama-3.1-8b-instant"
		case "4":
			return "mixtral-8x7b-32768"
		case "5":
			return "gemma2-9b-it"
		case "6":
			fmt.Print("Enter custom model name: ")
			model, _ := reader.ReadString('\n')
			return strings.TrimSpace(model)
		default:
			return "meta-llama/llama-4-maverick-17b-128e-instruct"
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [--provider PROVIDER] [PROMPT]")
		fmt.Println("\tgai config  # Configure default provider and model")
		fmt.Println("\tgai version # Show version")
		os.Exit(1)
	}

	if os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	if os.Args[1] == "config" {
		config, _ := loadConfig()
		reader := bufio.NewReader(os.Stdin)
		
		fmt.Println("\nConfigure default settings")
		fmt.Println("Current provider:", config.DefaultProvider)
		fmt.Println("Current model:", config.DefaultModel)
		
		fmt.Print("\nSelect default provider (1=groq, 2=openrouter): ")
		providerChoice, _ := reader.ReadString('\n')
		providerChoice = strings.TrimSpace(providerChoice)
		
		switch providerChoice {
		case "1":
			config.DefaultProvider = "groq"
		case "2":
			config.DefaultProvider = "openrouter"
		default:
			fmt.Println("Invalid choice, keeping current provider")
		}
		
		if providerChoice == "1" || providerChoice == "2" {
			model := getModelChoice(config.DefaultProvider)
			config.DefaultModel = model
			
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save configuration:", err)
			} else {
				fmt.Println("\nConfiguration saved!")
				fmt.Println("Default provider:", config.DefaultProvider)
				fmt.Println("Default model:", config.DefaultModel)
			}
		}
		
		os.Exit(0)
	}

	config, _ := loadConfig()
	
	// Use default provider from config if set, otherwise use "groq"
	defaultProvider := config.DefaultProvider
	if defaultProvider == "" {
		defaultProvider = "groq"
	}
	
	provider := defaultProvider

	// Setup flags after handling version check
	flag.StringVar(&provider, "provider", defaultProvider, "llm provider: groq or openrouter")
	flag.Parse()

	// Remaining args after flag parsing are the prompt tokens
	promptArgs := flag.Args()
	if len(promptArgs) == 0 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [--provider PROVIDER] [PROMPT]")
		fmt.Println("\tgai config  # Configure default provider and model")
		fmt.Println("\tgai version # Show version")
		os.Exit(1)
	}

	var apiKey string
	var model string
	needsModelConfig := false

	switch provider {
	case "openrouter":
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			apiKey = config.OpenRouterAPIKey
		}
		if apiKey == "" {
			fmt.Print("OPENROUTER_API_KEY is not set in env or config. Get one from https://openrouter.ai. Please enter your OPENROUTER_API_KEY: ")
			reader := bufio.NewReader(os.Stdin)
			inputAPIKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(inputAPIKey)

			if err := testAPIKey(provider, apiKey); err != nil {
				fmt.Println("OPENROUTER_API_KEY is invalid:", err)
				fmt.Println("Please check your API key or get a new one from https://openrouter.ai")
				os.Exit(1)
			}

			config.OpenRouterAPIKey = apiKey
			needsModelConfig = true
		}
		if apiKey == "" {
			fmt.Println("OPENROUTER_API_KEY is required. Get one from https://openrouter.ai")
			os.Exit(1)
		}
	default: // groq
		apiKey = os.Getenv("GROQ_API_KEY")
		if apiKey == "" {
			apiKey = config.GroqAPIKey
		}
		if apiKey == "" {
			fmt.Print("GROQ_API_KEY is not set in env or config. You can get one from https://console.groq.com/keys. Please enter your GROQ_API_KEY: ")
			reader := bufio.NewReader(os.Stdin)
			inputAPIKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(inputAPIKey)

			if err := testAPIKey(provider, apiKey); err != nil {
				fmt.Println("GROQ_API_KEY is invalid:", err)
				fmt.Println("Please check your API key or get a new one from https://console.groq.com/keys")
				os.Exit(1)
			}

			config.GroqAPIKey = apiKey
			needsModelConfig = true
		}
		if apiKey == "" {
			fmt.Println("GROQ_API_KEY is required. Get one from https://console.groq.com/keys")
			os.Exit(1)
		}
	}

	// Get model configuration
	if config.DefaultModel != "" && config.DefaultProvider == provider {
		model = config.DefaultModel
	} else if needsModelConfig || config.DefaultProvider != provider || config.DefaultModel == "" {
		// Ask for model choice when setting up new provider, provider changed, or model not configured
		model = getModelChoice(provider)
		config.DefaultProvider = provider
		config.DefaultModel = model
		if err := saveConfig(config); err != nil {
			fmt.Println("Failed to save configuration.")
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
