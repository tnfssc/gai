package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	GroqAPIKey string `json:"groq_api_key"`
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

func generateCommand(prompt string, apiKey string) (string, error) {
	llm, err := openai.New(
		openai.WithModel("llama-3.3-70b-specdec"),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
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

func testAPIKey(apiKey string) error {
	llm, err := openai.New(
		openai.WithModel("llama-3.3-70b-specdec"),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
		openai.WithToken(apiKey),
	)
	if err != nil {
		return err
	}

	_, err = llms.GenerateFromSinglePrompt(context.Background(), llm, "Say hello")
	return err
}

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
	config, _ := loadConfig()

	if apiKey == "" {
		if config.GroqAPIKey != "" {
			apiKey = config.GroqAPIKey
		} else {
			fmt.Print("GROQ_API_KEY is not set in env or config. You can get one from https://console.groq.com/keys. Please enter your GROQ_API_KEY: ")
			reader := bufio.NewReader(os.Stdin)
			inputAPIKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(inputAPIKey)

			if err := testAPIKey(apiKey); err != nil {
				fmt.Println("GROQ_API_KEY is invalid:", err)
				fmt.Println("Please check your API key or get a new one from https://console.groq.com/keys")
				os.Exit(1)
			}

			config.GroqAPIKey = apiKey
			if err := saveConfig(config); err != nil {
				fmt.Println("Failed to save GROQ_API_KEY to config file.")
			}
		}
	}

	if apiKey == "" {
		fmt.Println("GROQ_API_KEY is required. Get one from https://console.groq.com/keys")
		os.Exit(1)
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
`, shell, kernel, stdinText, strings.Join(os.Args[1:], " "))

	promptHash := hashPrompt(prompt)
	if cachedResponse, ok := loadCache(promptHash); ok {
		fmt.Println()
		fmt.Println("", cachedResponse)
		writeToClipboard(cachedResponse)
		return
	}

	completion, err := generateCommand(prompt, apiKey)
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
