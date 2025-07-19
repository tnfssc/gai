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
	KimiAPIKey      string `json:"kimi_api_key"`
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

func generateCommand(prompt, provider, apiKey string) (string, error) {
	var (
		model   string
		baseURL string
	)

	switch provider {
	case "kimi":
		model = "openrouter/moonshotai/kimi-k2"
		baseURL = "https://openrouter.ai/api/v1"
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
	case "kimi":
		model = "openrouter/moonshotai/kimi-k2"
		baseURL = "https://openrouter.ai/api/v1"
	case "openrouter":
		model = "deepseek/deepseek-chat:free"
		baseURL = "https://openrouter.ai/api/v1"
	default: // groq
		model = "llama-3.3-70b-specdec"
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

func runAgenticLoop(provider, apiKey string) {
	shell := getShell()
	kernel := runtime.GOOS
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("🤖 Agentic mode activated! I'm ready to help you with tasks.")
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println("Type 'clear' to clear the conversation history.")
	fmt.Println()

	var conversationHistory []string

	for {
		fmt.Print("💬 You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		if input == "clear" {
			conversationHistory = nil
			fmt.Println("🧹 Conversation history cleared.")
			continue
		}

		// Build the prompt with conversation history
		var historyText string
		if len(conversationHistory) > 0 {
			historyText = "\n<Conversation History>\n" + strings.Join(conversationHistory, "\n") + "\n</Conversation History>\n"
		}

		prompt := fmt.Sprintf(`
<Instructions Start>
You are an intelligent AI agent that can help users with various tasks. You have access to the shell environment and can execute commands.

Your capabilities include:
- File and directory operations
- System administration tasks
- Package management
- Network operations
- Text processing and analysis
- And much more

When the user asks for something that requires action, respond with the appropriate command(s) that can be executed in the %s shell on %s kernel.

If the user asks for information or explanation, provide a helpful response.

You can maintain context across multiple interactions. If the user refers to previous commands or results, use that context.

Always ensure commands are safe and appropriate for the context.

Respond naturally and conversationally, but be concise.
<Instructions End>
%s

<User Request>
%s
</User Request>
`, shell, kernel, historyText, input)

		fmt.Print("🤖 AI: ")
		
		completion, err := generateCommand(prompt, provider, apiKey)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		fmt.Println(completion)
		
		// Add to conversation history
		conversationHistory = append(conversationHistory, fmt.Sprintf("User: %s", input))
		conversationHistory = append(conversationHistory, fmt.Sprintf("AI: %s", completion))
		
		// Keep only last 10 exchanges to prevent context overflow
		if len(conversationHistory) > 20 {
			conversationHistory = conversationHistory[len(conversationHistory)-20:]
		}

		fmt.Println()
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [--provider PROVIDER] [--agentic] [PROMPT]")
		fmt.Println("        gai [--provider PROVIDER] --agentic")
		os.Exit(1)
	}

	if os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	provider := "groq"
	agenticMode := false

	// Setup flags after handling version check
	flag.StringVar(&provider, "provider", "groq", "llm provider: groq, openrouter, or kimi")
	flag.BoolVar(&agenticMode, "agentic", false, "enable agentic mode with interactive loop")
	flag.Parse()

	// Remaining args after flag parsing are the prompt tokens
	promptArgs := flag.Args()
	
	// If agentic mode is enabled and no prompt is provided, start the loop
	if agenticMode && len(promptArgs) == 0 {
		// Handle API key setup for agentic mode
		config, _ := loadConfig()
		var apiKey string

		switch provider {
		case "kimi":
			apiKey = os.Getenv("KIMI_API_KEY")
			if apiKey == "" {
				apiKey = config.KimiAPIKey
			}
			if apiKey == "" {
				fmt.Print("KIMI_API_KEY is not set in env or config. Get one from https://openrouter.ai. Please enter your KIMI_API_KEY: ")
				reader := bufio.NewReader(os.Stdin)
				inputAPIKey, _ := reader.ReadString('\n')
				apiKey = strings.TrimSpace(inputAPIKey)

				if err := testAPIKey(provider, apiKey); err != nil {
					fmt.Println("KIMI_API_KEY is invalid:", err)
					fmt.Println("Please check your API key or get a new one from https://openrouter.ai")
					os.Exit(1)
				}

				config.KimiAPIKey = apiKey
				if err := saveConfig(config); err != nil {
					fmt.Println("Failed to save KIMI_API_KEY to config file.")
				}
			}
			if apiKey == "" {
				fmt.Println("KIMI_API_KEY is required. Get one from https://openrouter.ai")
				os.Exit(1)
			}
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
				if err := saveConfig(config); err != nil {
					fmt.Println("Failed to save OPENROUTER_API_KEY to config file.")
				}
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
				if err := saveConfig(config); err != nil {
					fmt.Println("Failed to save GROQ_API_KEY to config file.")
				}
			}
			if apiKey == "" {
				fmt.Println("GROQ_API_KEY is required. Get one from https://console.groq.com/keys")
				os.Exit(1)
			}
		}

		runAgenticLoop(provider, apiKey)
		return
	}

	// Regular single-shot mode
	if len(promptArgs) == 0 {
		fmt.Println("Missing prompt")
		fmt.Println("Usage:\tgai [--provider PROVIDER] [--agentic] [PROMPT]")
		fmt.Println("        gai [--provider PROVIDER] --agentic")
		os.Exit(1)
	}

	config, _ := loadConfig()
	var apiKey string

	switch provider {
	case "kimi":
		apiKey = os.Getenv("KIMI_API_KEY")
		if apiKey == "" {
			apiKey = config.KimiAPIKey
		}
		if apiKey == "" {
			fmt.Print("KIMI_API_KEY is not set in env or config. Get one from https://openrouter.ai. Please enter your KIMI_API_KEY: ")
			reader := bufio.NewReader(os.Stdin)
			inputAPIKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(inputAPIKey)

			if err := testAPIKey(provider, apiKey); err != nil {
				fmt.Println("KIMI_API_KEY is invalid:", err)
				fmt.Println("Please check your API key or get a new one from https://openrouter.ai")
				os.Exit(1)
			}

			config.KimiAPIKey = apiKey
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save KIMI_API_KEY to config file.")
			}
		}
		if apiKey == "" {
			fmt.Println("KIMI_API_KEY is required. Get one from https://openrouter.ai")
			os.Exit(1)
		}
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
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save OPENROUTER_API_KEY to config file.")
			}
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
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save GROQ_API_KEY to config file.")
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
You are an intelligent AI agent that can help users with various tasks. You have access to the shell environment and can execute commands.

Your capabilities include:
- File and directory operations
- System administration tasks
- Package management
- Network operations
- Text processing and analysis
- And much more

When the user asks for something that requires action, respond with the appropriate command(s) that can be executed in the %s shell on %s kernel.

If the user asks for information or explanation, provide a helpful response.

Always ensure commands are safe and appropriate for the context.

Respond naturally and conversationally, but be concise.
<Instructions End>

%s

<User Request>
%s
</User Request>
`, shell, kernel, stdinText, strings.Join(promptArgs, " "))

	promptHash := hashPrompt(prompt)
	if cachedResponse, ok := loadCache(promptHash); ok {
		fmt.Println()
		fmt.Println("", cachedResponse)
		writeToClipboard(cachedResponse)
		return
	}

	completion, err := generateCommand(prompt, provider, apiKey)
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
