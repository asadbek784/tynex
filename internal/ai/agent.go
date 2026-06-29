package ai

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/asadbek_784/tynex-cli/internal/config"
	"github.com/asadbek_784/tynex-cli/internal/fileops"
	"github.com/asadbek_784/tynex-cli/internal/provider"
	"github.com/asadbek_784/tynex-cli/internal/shell"
	"github.com/asadbek_784/tynex-cli/internal/tui"
)

// Agent manages conversation with an AI provider and executes tool calls.
type Agent struct {
	config       *config.Config
	provider     provider.Provider
	messages     []provider.Message
	model        string
	systemPrompt string
	// ConfirmDangerous, if set, is called before executing dangerous operations.
	// The function receives a description of the operation and returns true if confirmed.
	// If nil, dangerous operations are auto-confirmed (e.g., in -p mode).
	ConfirmDangerous func(description string) bool
}

// NewAgent creates a new AI agent.
func NewAgent(cfg *config.Config, prov provider.Provider, systemPrompt string) *Agent {
	if systemPrompt == "" {
		systemPrompt = cfg.SystemPrompt
	}
	return &Agent{
		config:       cfg,
		provider:     prov,
		messages:     []provider.Message{},
		model:        cfg.Model,
		systemPrompt: systemPrompt,
	}
}

// AddMessage adds a message to the conversation history.
func (a *Agent) AddMessage(role, content string) {
	if content == "" {
		return
	}
	a.messages = append(a.messages, provider.Message{Role: role, Content: content})
}

// GetHistory returns the conversation history.
func (a *Agent) GetHistory() []provider.Message {
	return a.messages
}

// ClearHistory clears the conversation history.
func (a *Agent) ClearHistory() {
	a.messages = []provider.Message{}
}

// Tools returns the available tool definitions.
func (a *Agent) Tools() []provider.Tool {
	return []provider.Tool{
		{
			Name:        "read_file",
			Description: "Read the contents of a file at the given path. Returns the full file content as a string.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []interface{}{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Create a new file or overwrite an existing file with the given content. Creates intermediate directories as needed.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "File content to write",
					},
				},
				"required": []interface{}{"path", "content"},
			},
		},
		{
			Name:        "str_replace",
			Description: "Replace an exact string match in a file with a new string. Use this for targeted edits rather than rewriting entire files.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"old_string": map[string]interface{}{
						"type":        "string",
						"description": "The exact string to replace (must match exactly, including whitespace)",
					},
					"new_string": map[string]interface{}{
						"type":        "string",
						"description": "The new string to insert",
					},
				},
				"required": []interface{}{"path", "old_string", "new_string"},
			},
		},
		{
			Name:        "code_search",
			Description: "Search the codebase for a pattern using ripgrep or grep. Returns matching lines with file paths and line numbers.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Search pattern (regex supported)",
					},
				},
				"required": []interface{}{"pattern"},
			},
		},
		{
			Name:        "list_directory",
			Description: "List files and subdirectories in a directory path. Returns separate lists of files and directories.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to list",
					},
				},
				"required": []interface{}{"path"},
			},
		},
		{
			Name:        "run_terminal",
			Description: "Execute a shell command on the user's terminal and capture its output. Use this for running tests, builds, linters, git commands, or any terminal operations. Be careful with destructive commands.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Shell command to execute",
					},
				},
				"required": []interface{}{"command"},
			},
		},
		{
			Name:        "ask_user",
			Description: "Ask the user a question when you need guidance, clarification, or a decision. Use this when you're unsure about something important.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question": map[string]interface{}{
						"type":        "string",
						"description": "Question to ask the user",
					},
				},
				"required": []interface{}{"question"},
			},
		},
		{
			Name:        "glob",
			Description: "Find files matching a glob pattern (e.g., **/*.go, src/**/*.ts). Returns file paths sorted by modification time.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Glob pattern to match (e.g., **/*.ts, **/test_*.go)",
					},
				},
				"required": []interface{}{"pattern"},
			},
		},
	}
}

// confirm asks the user for confirmation of a dangerous operation.
// Returns true if the user confirms, false if denied.
func (a *Agent) confirm(description string) bool {
	if a.ConfirmDangerous != nil {
		return a.ConfirmDangerous(description)
	}
	// Auto-confirm in non-interactive mode
	return true
}

// ConfirmInteractive asks for user confirmation via stdin.
func ConfirmInteractive(description string) bool {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║         ⚠️  XAVFSIZLIK TEKSHIRUVI        ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println(description)
	fmt.Println()
	fmt.Print("Davom etishni tasdiqlaysizmi? (y/N): ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}

// showDiff shows a preview of file changes (for write_file operations).
func showDiff(path, newContent string) string {
	existingContent, err := fileops.ReadFile(path)
	var preview string
	if err != nil {
		// File doesn't exist yet — show it as new
		preview = fmt.Sprintf("📄 Yangi fayl: %s\n", path)
		if len(newContent) > 1000 {
			preview += fmt.Sprintf("   %d bayt (quyida 1000 belgi ko'rsatilgan)\n\n", len(newContent))
			preview += newContent[:1000] + "...\n"
		} else {
			preview += newContent + "\n"
		}
	} else {
		// File exists — show a simple diff
		preview = fmt.Sprintf("📝 Fayl o'zgarishi: %s\n", path)
		if existingContent == newContent {
			preview += "   (hech qanday o'zgarish yo'q)\n"
		} else {
			preview += fmt.Sprintf("   Eski: %d bayt → Yangi: %d bayt\n", len(existingContent), len(newContent))
			if len(newContent) > 1000 {
				preview += fmt.Sprintf("   Yangi mazmun (boshi, 1000 belgi):\n\n%s...\n", newContent[:1000])
			} else {
				preview += fmt.Sprintf("   Yangi mazmun:\n\n%s\n", newContent)
			}
		}
	}
	return preview
}

// ExecuteTool executes a tool call and returns the result as a string.
func (a *Agent) ExecuteTool(toolCall provider.ToolCall) string {
	switch toolCall.Name {
	case "read_file":
		path, _ := toolCall.Arguments["path"].(string)
		if path == "" {
			return "Error: path is required"
		}
		content, err := fileops.ReadFile(path)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return content

	case "write_file":
		path, _ := toolCall.Arguments["path"].(string)
		content, _ := toolCall.Arguments["content"].(string)
		if path == "" {
			return "Error: path is required"
		}

		description := showDiff(path, content)
		if !a.confirm(description) {
			return fmt.Sprintf("⛔ Bekor qilindi: fayl yozilmadi (%s)", path)
		}

		if err := fileops.WriteFile(path, content); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return fmt.Sprintf("Successfully wrote file: %s (%d bytes)", path, len(content))

	case "str_replace":
		path, _ := toolCall.Arguments["path"].(string)
		oldStr, _ := toolCall.Arguments["old_string"].(string)
		newStr, _ := toolCall.Arguments["new_string"].(string)
		if path == "" || oldStr == "" {
			return "Error: path and old_string are required"
		}

		description := fmt.Sprintf("🔧 Matn almashtirish: %s\n\n   Qidirilayotgan matn:\n   %q\n\n   Yangi matn:\n   %q\n", path, oldStr, newStr)
		if !a.confirm(description) {
			return fmt.Sprintf("⛔ Bekor qilindi: matn almashtirilmadi (%s)", path)
		}

		count, err := fileops.StrReplace(path, oldStr, newStr)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return fmt.Sprintf("Successfully made %d replacement(s) in %s", count, path)

	case "code_search":
		pattern, _ := toolCall.Arguments["pattern"].(string)
		if pattern == "" {
			return "Error: pattern is required"
		}
		results, err := fileops.CodeSearch(pattern)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		if len(results) == 0 {
			return fmt.Sprintf("No results found for pattern: %s", pattern)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d result(s) for pattern '%s':\n\n", len(results), pattern))
		for _, r := range results {
			sb.WriteString(fmt.Sprintf("%s:%d: %s\n", r.Path, r.Line, r.Content))
		}
		return sb.String()

	case "list_directory":
		path, _ := toolCall.Arguments["path"].(string)
		if path == "" {
			path = "."
		}
		files, dirs, err := fileops.ListDirectory(path)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Directory: %s\n", path))
		if len(dirs) > 0 {
			sb.WriteString("\nDirectories:\n")
			for _, d := range dirs {
				sb.WriteString(fmt.Sprintf("  📁 %s/\n", d))
			}
		}
		if len(files) > 0 {
			sb.WriteString("\nFiles:\n")
			for _, f := range files {
				sb.WriteString(fmt.Sprintf("  📄 %s\n", f))
			}
		}
		if len(dirs) == 0 && len(files) == 0 {
			sb.WriteString("  (empty directory)\n")
		}
		return sb.String()

	case "run_terminal":
		command, _ := toolCall.Arguments["command"].(string)
		if command == "" {
			return "Error: command is required"
		}

		description := fmt.Sprintf("💻 Terminal buyrug'i:\n\n   $ %s\n", command)
		if !a.confirm(description) {
			return fmt.Sprintf("⛔ Bekor qilindi: buyruq bajarilmadi")
		}

		result, err := shell.RunCommand(command, 30)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Command: %s\n", command))
		sb.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
		sb.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration))
		if result.Stdout != "" {
			sb.WriteString(fmt.Sprintf("\nStdout:\n%s\n", result.Stdout))
		}
		if result.Stderr != "" {
			sb.WriteString(fmt.Sprintf("\nStderr:\n%s\n", result.Stderr))
		}
		return sb.String()

	case "ask_user":
		question, _ := toolCall.Arguments["question"].(string)
		if question == "" {
			return "Error: question is required"
		}
		// Return the question so the chat loop can handle user interaction
		return fmt.Sprintf("[ASK_USER]: %s\n(Pending user response...)", question)

	case "glob":
		pattern, _ := toolCall.Arguments["pattern"].(string)
		if pattern == "" {
			return "Error: pattern is required"
		}
		matches, err := fileops.Glob(pattern)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		if len(matches) == 0 {
			return fmt.Sprintf("No files match pattern: %s", pattern)
		}
		return fmt.Sprintf("Found %d file(s):\n%s", len(matches), strings.Join(matches, "\n"))

	default:
		return fmt.Sprintf("Unknown tool: %s", toolCall.Name)
	}
}

// SetProvider changes the underlying AI provider (used by fallback system).
func (a *Agent) SetProvider(prov provider.Provider) {
	a.provider = prov
}

// SendMessageWithFallback tries the primary provider first, then falls back to others.
func (a *Agent) SendMessageWithFallback(ctx context.Context, userInput string, onStream func(string), fallbackProviders []FallbackProvider) (*provider.Response, error) {
	a.AddMessage("user", userInput)

	// Try primary provider
	resp, err := a.sendMessageInternal(ctx, onStream)
	if err == nil {
		return resp, nil
	}

	// Primary failed - try fallbacks
	fmt.Fprintf(os.Stderr, "\n⚠️  Asosiy provayder xatosi: %v\n", err)

	for _, fb := range fallbackProviders {
		fmt.Fprintf(os.Stderr, "\n🔄 Fallback: %s provayderiga o'tilmoqda...\n", fb.Name)

		a.SetProvider(fb.Provider)
		a.model = fb.Model

		// Remove the last user message before retry
		if len(a.messages) > 0 && a.messages[len(a.messages)-1].Role == "user" {
			a.messages = a.messages[:len(a.messages)-1]
		}

		resp, err := a.sendMessageInternal(ctx, onStream)
		if err == nil {
			fmt.Fprintf(os.Stderr, "✅ Fallback %s muvaffaqiyatli\n", fb.Name)
			return resp, nil
		}
		fmt.Fprintf(os.Stderr, "❌ Fallback %s ham xato: %v\n", fb.Name, err)
	}

	return nil, fmt.Errorf("barcha provayderlar muvaffaqiyatsiz: %w", err)
}

// sendMessageInternal is the core message sending loop used by SendMessage and SendMessageWithFallback.
func (a *Agent) sendMessageInternal(ctx context.Context, onStream func(string)) (*provider.Response, error) {
	req := &provider.Request{
		Messages:    a.messages,
		System:      a.systemPrompt,
		Model:       a.model,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Tools:       a.Tools(),
		Stream:      false,
	}

	for i := 0; i < 20; i++ {
		resp, err := a.provider.SendMessage(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("AI request failed: %w", err)
		}

		if len(resp.ToolCalls) == 0 {
			a.AddMessage("assistant", resp.Content)
			if onStream != nil && resp.Content != "" {
				onStream(resp.Content)
			}
			return resp, nil
		}

	if resp.Content != "" {
		a.AddMessage("assistant", resp.Content)
	}

	for _, tc := range resp.ToolCalls {
		if onStream != nil {
			// Format tool call using TUI
			argsStr := formatToolArgs(tc.Arguments)
			onStream(tui.RenderToolCall(tc.Name, argsStr) + "\n")
		}

		result := a.ExecuteTool(tc)

		if onStream != nil {
			onStream(fmt.Sprintf("\n📝 Result: %s\n\n", truncate(result, 500)))
		}

		a.messages = append(a.messages, provider.Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool '%s' returned:\n%s", tc.Name, result),
		})
	}

	req = &provider.Request{
		Messages:    a.messages,
		System:      a.systemPrompt,
		Model:       a.model,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Tools:       a.Tools(),
		Stream:      false,
	}
	}

	return nil, fmt.Errorf("tool call limit exceeded (20 iterations)")
}

// SendMessage sends a message to the AI and processes the response,
// including tool calls, until a text response is received.
func (a *Agent) SendMessage(ctx context.Context, userInput string, onStream func(string)) (*provider.Response, error) {
	a.AddMessage("user", userInput)
	return a.sendMessageInternal(ctx, onStream)
}

// SendMessageStream sends a message with streaming response.
func (a *Agent) SendMessageStream(ctx context.Context, userInput string, onChunk func(string), onTool func(provider.ToolCall), onToolResult func(string, string)) (*provider.Response, error) {
	a.AddMessage("user", userInput)

	req := &provider.Request{
		Messages:    a.messages,
		System:      a.systemPrompt,
		Model:       a.model,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Tools:       a.Tools(),
		Stream:      true,
	}

	for i := 0; i < 20; i++ {
		// Non-streaming first to handle tool calls reliably
		req.Stream = false
		resp, err := a.provider.SendMessage(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("AI request failed: %w", err)
		}

		if len(resp.ToolCalls) == 0 {
			if resp.Content != "" {
				a.AddMessage("assistant", resp.Content)
				if onChunk != nil {
					onChunk(resp.Content)
				}
			}
			return resp, nil
		}

		// Store assistant response
		// Store assistant response
		if resp.Content != "" {
			a.AddMessage("assistant", resp.Content)
		}

		// Process tools
		for _, tc := range resp.ToolCalls {
			if onTool != nil {
				onTool(tc)
			}
			// Log tool call via TUI
			if onChunk != nil {
				argsStr := formatToolArgs(tc.Arguments)
				onChunk("\n" + tui.RenderToolCall(tc.Name, argsStr) + "\n")
			}
			result := a.ExecuteTool(tc)
			if onToolResult != nil {
				onToolResult(tc.Name, result)
			}

			a.messages = append(a.messages, provider.Message{
				Role:    "user",
				Content: fmt.Sprintf("Tool '%s' returned:\n%s", tc.Name, result),
			})
		}

		req = &provider.Request{
			Messages:    a.messages,
			System:      a.systemPrompt,
			Model:       a.model,
			MaxTokens:   a.config.MaxTokens,
			Temperature: a.config.Temperature,
			Tools:       a.Tools(),
			Stream:      false,
		}
	}

	return nil, fmt.Errorf("tool call limit exceeded (20 iterations)")
}

// FallbackProvider holds a fallback provider configuration for the fallback system.
type FallbackProvider struct {
	Name     string
	Provider provider.Provider
	Model    string
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatToolArgs formats tool call arguments into a readable string.
func formatToolArgs(args map[string]interface{}) string {
	var parts []string
	for k, v := range args {
		switch val := v.(type) {
		case string:
			runes := []rune(val)
			if len(runes) > 60 {
				parts = append(parts, fmt.Sprintf("%s=%q...", k, string(runes[:60])))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%q", k, val))
			}
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", k, val))
		}
	}
	return strings.Join(parts, ", ")
}


