package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"ai-chat/internal/agent"
	"ai-chat/internal/config"
	"ai-chat/internal/models"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	configFile       string
	systemPromptFile string
	messageWindow    int
	modelFlag        string
	openaiBaseURL    string
	anthropicBaseURL string
	openaiAPIKey     string
	anthropicAPIKey  string
	googleAPIKey     string
	debugMode        bool
	promptFlag       string
	quietFlag        bool
	scriptFlag       bool
	maxSteps         int
	scriptMCPConfig  *config.Config // Used to override config in script mode
)

var rootCmd = &cobra.Command{
	Use:   "mcphost",
	Short: "Chat with AI models through a unified interface",
	Long: `MCPHost is a CLI tool that allows you to interact with various AI models
through a unified interface. It supports various tools through MCP servers
and provides streaming responses.

Available models can be specified using the --model flag:
- Anthropic Claude (default): anthropic:claude-sonnet-4-20250514
- OpenAI: openai:gpt-4
- Ollama models: ollama:modelname
- Google: google:modelname

Examples:
  # Interactive mode
  mcphost -m ollama:qwen2.5:3b
  mcphost -m openai:gpt-4
  mcphost -m google:gemini-2.0-flash

  # Non-interactive mode
  mcphost -p "What is the weather like today?"
  mcphost -p "Calculate 15 * 23" --quiet

  # Script mode
  mcphost --script myscript.sh
  ./myscript.sh  # if script has shebang #!/path/to/mcphost --script`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPHost(context.Background())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().
		StringVar(&configFile, "config", "", "config file (default is $HOME/.mcp.json)")
	rootCmd.PersistentFlags().
		StringVar(&systemPromptFile, "system-prompt", "", "system prompt text or path to system prompt json file")
	rootCmd.PersistentFlags().
		IntVar(&messageWindow, "message-window", 40, "number of messages to keep in context")
	rootCmd.PersistentFlags().
		StringVarP(&modelFlag, "model", "m", "anthropic:claude-sonnet-4-20250514",
			"model to use (format: provider:model)")
	rootCmd.PersistentFlags().
		BoolVar(&debugMode, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().
		StringVarP(&promptFlag, "prompt", "p", "", "run in non-interactive mode with the given prompt")
	rootCmd.PersistentFlags().
		BoolVar(&quietFlag, "quiet", false, "suppress all output (only works with --prompt)")
	rootCmd.PersistentFlags().
		BoolVar(&scriptFlag, "script", false, "run in script mode (parse YAML frontmatter and prompt from file)")
	rootCmd.PersistentFlags().
		IntVar(&maxSteps, "max-steps", 0, "maximum number of agent steps (0 for unlimited)")

	flags := rootCmd.PersistentFlags()
	flags.StringVar(&openaiBaseURL, "openai-url", "", "base URL for OpenAI API")
	flags.StringVar(&anthropicBaseURL, "anthropic-url", "", "base URL for Anthropic API")
	flags.StringVar(&openaiAPIKey, "openai-api-key", "", "OpenAI API key")
	flags.StringVar(&anthropicAPIKey, "anthropic-api-key", "", "Anthropic API key")
	flags.StringVar(&googleAPIKey, "google-api-key", "", "Google (Gemini) API key")

	// Bind flags to viper for config file support
	viper.BindPFlag("system-prompt", rootCmd.PersistentFlags().Lookup("system-prompt"))
	viper.BindPFlag("message-window", rootCmd.PersistentFlags().Lookup("message-window"))
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("max-steps", rootCmd.PersistentFlags().Lookup("max-steps"))
	viper.BindPFlag("openai-url", rootCmd.PersistentFlags().Lookup("openai-url"))
	viper.BindPFlag("anthropic-url", rootCmd.PersistentFlags().Lookup("anthropic-url"))
	viper.BindPFlag("openai-api-key", rootCmd.PersistentFlags().Lookup("openai-api-key"))
	viper.BindPFlag("anthropic-api-key", rootCmd.PersistentFlags().Lookup("anthropic-api-key"))
	viper.BindPFlag("google-api-key", rootCmd.PersistentFlags().Lookup("google-api-key"))
}

func runMCPHost(ctx context.Context) error {
	// Handle script mode
	if scriptFlag {
		return runScriptMode(ctx)
	}

	return runNormalMode(ctx)
}

func runNormalMode(ctx context.Context) error {
	// Validate flag combinations
	if quietFlag && promptFlag == "" {
		return fmt.Errorf("--quiet flag can only be used with --prompt/-p")
	}

	// Set up logging
	if debugMode {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Load configuration
	var mcpConfig *config.Config
	var err error

	if scriptMCPConfig != nil {
		// Use script-provided config
		mcpConfig = scriptMCPConfig
	} else {
		// Load normal config
		mcpConfig, err = config.LoadMCPConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load MCP config: %v", err)
		}
	}

	// Set up viper to read from the same config file for flag values
	if configFile == "" {
		// Use default config file locations
		homeDir, err := os.UserHomeDir()
		if err == nil {
			viper.SetConfigName(".mcphost")
			viper.AddConfigPath(homeDir)
			viper.SetConfigType("yaml")
			if err := viper.ReadInConfig(); err != nil {
				// Try .mcphost.json
				viper.SetConfigType("json")
				if err := viper.ReadInConfig(); err != nil {
					// Try legacy .mcp files
					viper.SetConfigName(".mcp")
					viper.SetConfigType("yaml")
					if err := viper.ReadInConfig(); err != nil {
						viper.SetConfigType("json")
						viper.ReadInConfig() // Ignore error if no config found
					}
				}
			}
		}
	} else {
		// Use specified config file
		viper.SetConfigFile(configFile)
		viper.ReadInConfig() // Ignore error if file doesn't exist
	}

	// Override flag values with config file values (using viper's bound values)
	if viper.GetString("system-prompt") != "" {
		systemPromptFile = viper.GetString("system-prompt")
	}
	if viper.GetInt("message-window") != 0 {
		messageWindow = viper.GetInt("message-window")
	}
	if viper.GetString("model") != "" {
		modelFlag = viper.GetString("model")
	}
	if viper.GetBool("debug") {
		debugMode = viper.GetBool("debug")
	}
	if viper.GetInt("max-steps") != 0 {
		maxSteps = viper.GetInt("max-steps")
	}
	if viper.GetString("openai-url") != "" {
		openaiBaseURL = viper.GetString("openai-url")
	}
	if viper.GetString("anthropic-url") != "" {
		anthropicBaseURL = viper.GetString("anthropic-url")
	}
	if viper.GetString("openai-api-key") != "" {
		openaiAPIKey = viper.GetString("openai-api-key")
	}
	if viper.GetString("anthropic-api-key") != "" {
		anthropicAPIKey = viper.GetString("anthropic-api-key")
	}
	if viper.GetString("google-api-key") != "" {
		googleAPIKey = viper.GetString("google-api-key")
	}

	systemPrompt, err := config.LoadSystemPrompt(systemPromptFile)
	if err != nil {
		return fmt.Errorf("failed to load system prompt: %v", err)
	}

	// Create model configuration
	modelConfig := &models.ProviderConfig{
		ModelString:      modelFlag,
		SystemPrompt:     systemPrompt,
		AnthropicAPIKey:  anthropicAPIKey,
		AnthropicBaseURL: anthropicBaseURL,
		OpenAIAPIKey:     openaiAPIKey,
		OpenAIBaseURL:    openaiBaseURL,
		GoogleAPIKey:     googleAPIKey,
	}

	// Create agent configuration
	agentMaxSteps := maxSteps
	if agentMaxSteps == 0 {
		agentMaxSteps = 1000 // Set a high limit for "unlimited"
	}

	agentConfig := &agent.AgentConfig{
		ModelConfig:   modelConfig,
		MCPConfig:     mcpConfig,
		SystemPrompt:  systemPrompt,
		MaxSteps:      agentMaxSteps,
		MessageWindow: messageWindow,
	}

	// Create the agent
	mcpAgent, err := agent.NewAgent(ctx, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to create agent: %v", err)
	}
	defer mcpAgent.Close()

	// Get model name for display
	parts := strings.SplitN(modelFlag, ":", 2)
	modelName := "Unknown"
	if len(parts) == 2 {
		modelName = parts[1]
	}

	// Get tools
	tools := mcpAgent.GetTools()

	// Log initialization info
	if !quietFlag {
		fmt.Printf("Model loaded: %s\n", modelName)
		fmt.Printf("Loaded %d tools from MCP servers\n", len(tools))
	}

	// Prepare data for slash commands
	var serverNames []string
	for name := range mcpConfig.MCPServers {
		serverNames = append(serverNames, name)
	}

	var toolNames []string
	for _, tool := range tools {
		if info, err := tool.Info(ctx); err == nil {
			toolNames = append(toolNames, info.Name)
		}
	}

	// Main interaction logic
	var messages []*schema.Message

	// Check if running in non-interactive mode
	if promptFlag != "" {
		// Display user message (skip if quiet)
		if !quietFlag {
			fmt.Printf("You: %s\n", promptFlag)
		}

		// Add user message to history
		messages = append(messages, schema.UserMessage(promptFlag))

		// Get agent response
		response, err := mcpAgent.GenerateWithLoop(ctx, messages,
			// Tool call handler - called when a tool is about to be executed
			func(toolName, toolArgs string) {
				if !quietFlag {
					fmt.Printf("Calling tool: %s with args: %s\n", toolName, toolArgs)
				}
			},
			// Tool execution handler - called when tool execution starts/ends
			func(toolName string, isStarting bool) {
				if !quietFlag {
					if isStarting {
						fmt.Printf("Executing %s...\n", toolName)
					}
				}
			},
			// Tool result handler - called when a tool execution completes
			func(toolName, toolArgs, result string, isError bool) {
				if !quietFlag {
					if isError {
						fmt.Printf("Tool %s error: %s\n", toolName, result)
					} else {
						fmt.Printf("Tool %s result: %s\n", toolName, result)
					}
				}
			},
			// Response handler - called when the LLM generates a response
			func(content string) {
				// Nothing to do here
			},
			// Tool call content handler - called when content accompanies tool calls
			func(content string) {
				if !quietFlag {
					fmt.Printf("%s (%s): %s\n", modelName, "Assistant", content)
				}
			},
		)

		if err != nil {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			return err
		}

		// Display assistant response with model name (skip if quiet)
		if !quietFlag {
			fmt.Printf("%s (%s): %s\n", modelName, "Assistant", response.Content)
		} else {
			// In quiet mode, only output the final response content to stdout
			fmt.Print(response.Content)
		}

		return nil
	}

	// Quiet mode is not allowed in interactive mode
	if quietFlag {
		return fmt.Errorf("--quiet flag can only be used with --prompt/-p")
	}

	return runInteractiveMode(ctx, mcpAgent, serverNames, toolNames, modelName, messages)
}

// runInteractiveMode handles the interactive mode execution
func runInteractiveMode(ctx context.Context, mcpAgent *agent.Agent, serverNames, toolNames []string, modelName string, messages []*schema.Message) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to MCPHost. Type your message or use commands:")
	fmt.Println("  /help - Show help")
	fmt.Println("  /quit - Exit the application")
	fmt.Println("  /tools - List available tools")
	fmt.Println("  /servers - List configured MCP servers")
	fmt.Println("  /history - Display conversation history")
	fmt.Println("  /clear - Clear the screen")

	// Main interaction loop
	for {
		// Get user input
		fmt.Print("\nYou: ")
		prompt, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return nil
			}
			return fmt.Errorf("failed to get prompt: %v", err)
		}

		// Trim whitespace
		prompt = strings.TrimSpace(prompt)

		if prompt == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(prompt, "/") {
			switch prompt {
			case "/help":
				fmt.Println("\nAvailable Commands:")
				fmt.Println("  /help - Show this help message")
				fmt.Println("  /tools - List all available tools")
				fmt.Println("  /servers - List configured MCP servers")
				fmt.Println("  /history - Display conversation history")
				fmt.Println("  /clear - Clear the screen")
				fmt.Println("  /quit - Exit the application")
				continue
			case "/tools":
				fmt.Println("\nAvailable Tools:")
				if len(toolNames) == 0 {
					fmt.Println("  No tools are currently available.")
				} else {
					for i, tool := range toolNames {
						fmt.Printf("  %d. %s\n", i+1, tool)
					}
				}
				continue
			case "/servers":
				fmt.Println("\nConfigured MCP Servers:")
				if len(serverNames) == 0 {
					fmt.Println("  No MCP servers are currently configured.")
				} else {
					for i, server := range serverNames {
						fmt.Printf("  %d. %s\n", i+1, server)
					}
				}
				continue
			case "/history":
				fmt.Println("\nConversation History:")
				for i, msg := range messages {
					switch msg.Role {
					case schema.User:
						fmt.Printf("  You: %s\n", msg.Content)
					case schema.Assistant:
						fmt.Printf("  %s: %s\n", modelName, msg.Content)
					}
					if i < len(messages)-1 {
						fmt.Println()
					}
				}
				continue
			case "/clear":
				// Clear screen using ANSI escape code
				fmt.Print("\033[H\033[2J")
				continue
			case "/quit":
				fmt.Println("\nGoodbye!")
				return nil
			default:
				fmt.Printf("Unknown command: %s\n", prompt)
				continue
			}
		}

		// Add user message to history
		messages = append(messages, schema.UserMessage(prompt))

		// Prune messages if needed
		if len(messages) > messageWindow {
			messages = messages[len(messages)-messageWindow:]
		}

		// Get agent response
		fmt.Println("\nThinking...")

		response, err := mcpAgent.GenerateWithLoop(ctx, messages,
			// Tool call handler - called when a tool is about to be executed
			func(toolName, toolArgs string) {
				fmt.Printf("Calling tool: %s with args: %s\n", toolName, toolArgs)
			},
			// Tool execution handler - called when tool execution starts/ends
			func(toolName string, isStarting bool) {
				if isStarting {
					fmt.Printf("Executing %s...\n", toolName)
				}
			},
			// Tool result handler - called when a tool execution completes
			func(toolName, toolArgs, result string, isError bool) {
				if isError {
					fmt.Printf("Tool %s error: %s\n", toolName, result)
				} else {
					fmt.Printf("Tool %s result: %s\n", toolName, result)
				}
				fmt.Println("\nThinking...")
			},
			// Response handler - called when the LLM generates a response
			func(content string) {
				// Nothing to do here
			},
			// Tool call content handler - called when content accompanies tool calls
			func(content string) {
				fmt.Printf("\n%s (%s): %s\n", modelName, "Assistant", content)
				fmt.Println("\nThinking...")
			},
		)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		// Display assistant response with model name
		fmt.Printf("\n%s (%s): %s\n", modelName, "Assistant", response.Content)

		// Add assistant response to history
		messages = append(messages, response)
	}
}

// runScriptMode handles script mode execution
func runScriptMode(ctx context.Context) error {
	var scriptFile string

	// Determine script file from arguments
	// When called via shebang, the script file is the first non-flag argument
	// When called with --script flag, we need to find the script file in args
	args := os.Args[1:]

	// Filter out flags to find the script file
	for _, arg := range args {
		if arg == "--script" {
			// Skip the --script flag itself
			continue
		}
		if strings.HasPrefix(arg, "-") {
			// Skip other flags
			continue
		}
		// This should be our script file
		scriptFile = arg
		break
	}

	if scriptFile == "" {
		return fmt.Errorf("script mode requires a script file argument")
	}

	// Parse the script file
	scriptConfig, err := parseScriptFile(scriptFile)
	if err != nil {
		return fmt.Errorf("failed to parse script file: %v", err)
	}

	// Override the global configFile and promptFlag with script values
	originalConfigFile := configFile
	originalPromptFlag := promptFlag
	originalModelFlag := modelFlag
	originalMaxSteps := maxSteps
	originalMessageWindow := messageWindow
	originalDebugMode := debugMode
	originalSystemPromptFile := systemPromptFile
	originalOpenAIAPIKey := openaiAPIKey
	originalAnthropicAPIKey := anthropicAPIKey
	originalGoogleAPIKey := googleAPIKey
	originalOpenAIURL := openaiBaseURL
	originalAnthropicURL := anthropicBaseURL

	// Create config from script or load normal config
	var mcpConfig *config.Config
	if len(scriptConfig.MCPServers) > 0 {
		// Use servers from script
		mcpConfig = scriptConfig
	} else {
		// Fall back to normal config loading
		mcpConfig, err = config.LoadMCPConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load MCP config: %v", err)
		}
		// Merge script config values into loaded config
		if scriptConfig.Model != "" {
			mcpConfig.Model = scriptConfig.Model
		}
		if scriptConfig.MaxSteps != 0 {
			mcpConfig.MaxSteps = scriptConfig.MaxSteps
		}
		if scriptConfig.MessageWindow != 0 {
			mcpConfig.MessageWindow = scriptConfig.MessageWindow
		}
		if scriptConfig.Debug {
			mcpConfig.Debug = scriptConfig.Debug
		}
		if scriptConfig.SystemPrompt != "" {
			mcpConfig.SystemPrompt = scriptConfig.SystemPrompt
		}
		if scriptConfig.OpenAIAPIKey != "" {
			mcpConfig.OpenAIAPIKey = scriptConfig.OpenAIAPIKey
		}
		if scriptConfig.AnthropicAPIKey != "" {
			mcpConfig.AnthropicAPIKey = scriptConfig.AnthropicAPIKey
		}
		if scriptConfig.GoogleAPIKey != "" {
			mcpConfig.GoogleAPIKey = scriptConfig.GoogleAPIKey
		}
		if scriptConfig.OpenAIURL != "" {
			mcpConfig.OpenAIURL = scriptConfig.OpenAIURL
		}
		if scriptConfig.AnthropicURL != "" {
			mcpConfig.AnthropicURL = scriptConfig.AnthropicURL
		}
		if scriptConfig.Prompt != "" {
			mcpConfig.Prompt = scriptConfig.Prompt
		}
	}

	// Override the global config for normal mode
	scriptMCPConfig = mcpConfig

	// Apply script configuration to global flags
	if mcpConfig.Prompt != "" {
		promptFlag = mcpConfig.Prompt
	}
	if mcpConfig.Model != "" {
		modelFlag = mcpConfig.Model
	}
	if mcpConfig.MaxSteps != 0 {
		maxSteps = mcpConfig.MaxSteps
	}
	if mcpConfig.MessageWindow != 0 {
		messageWindow = mcpConfig.MessageWindow
	}
	if mcpConfig.Debug {
		debugMode = mcpConfig.Debug
	}
	if mcpConfig.SystemPrompt != "" {
		systemPromptFile = mcpConfig.SystemPrompt
	}
	if mcpConfig.OpenAIAPIKey != "" {
		openaiAPIKey = mcpConfig.OpenAIAPIKey
	}
	if mcpConfig.AnthropicAPIKey != "" {
		anthropicAPIKey = mcpConfig.AnthropicAPIKey
	}
	if mcpConfig.GoogleAPIKey != "" {
		googleAPIKey = mcpConfig.GoogleAPIKey
	}
	if mcpConfig.OpenAIURL != "" {
		openaiBaseURL = mcpConfig.OpenAIURL
	}
	if mcpConfig.AnthropicURL != "" {
		anthropicBaseURL = mcpConfig.AnthropicURL
	}

	// Restore original values after execution
	defer func() {
		configFile = originalConfigFile
		promptFlag = originalPromptFlag
		modelFlag = originalModelFlag
		maxSteps = originalMaxSteps
		messageWindow = originalMessageWindow
		debugMode = originalDebugMode
		systemPromptFile = originalSystemPromptFile
		openaiAPIKey = originalOpenAIAPIKey
		anthropicAPIKey = originalAnthropicAPIKey
		googleAPIKey = originalGoogleAPIKey
		openaiBaseURL = originalOpenAIURL
		anthropicBaseURL = originalAnthropicURL
		scriptMCPConfig = nil
	}()

	// Now run the normal execution path which will use our overridden config
	return runNormalMode(ctx)
}

// parseScriptFile parses a script file with YAML frontmatter and returns config
func parseScriptFile(filename string) (*config.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip shebang line if present
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#!") {
			// If it's not a shebang, we need to process this line
			return parseScriptContent(line + "\n" + readRemainingLines(scanner))
		}
	}

	// Read the rest of the file
	content := readRemainingLines(scanner)
	return parseScriptContent(content)
}

// readRemainingLines reads all remaining lines from a scanner
func readRemainingLines(scanner *bufio.Scanner) string {
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}

// parseScriptContent parses the content to extract YAML frontmatter
func parseScriptContent(content string) (*config.Config, error) {
	lines := strings.Split(content, "\n")

	// Find YAML frontmatter
	var yamlLines []string

	for _, line := range lines {
		yamlLines = append(yamlLines, line)
	}

	// Parse YAML
	yamlContent := strings.Join(yamlLines, "\n")
	var scriptConfig config.Config
	if err := yaml.Unmarshal([]byte(yamlContent), &scriptConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	return &scriptConfig, nil
}
