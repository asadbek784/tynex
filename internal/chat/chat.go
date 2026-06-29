package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asadbek_784/tynex-cli/internal/ai"
	"github.com/asadbek_784/tynex-cli/internal/config"
	"github.com/asadbek_784/tynex-cli/internal/provider"
	"github.com/asadbek_784/tynex-cli/internal/tui"
)

// ErrExit is sentinel error for clean chat exit.
var ErrExit = errors.New("exit")

// Session represents a saved chat session.
type Session struct {
	ID        string             `json:"id"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Messages  []provider.Message `json:"messages"`
	Provider  string             `json:"provider"`
	Model     string             `json:"model"`
}

// SessionStore manages saved chat sessions.
type SessionStore struct {
	sessionsDir string
}

// NewSessionStore creates a new session store.
func NewSessionStore() (*SessionStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home directory not found: %w", err)
	}
	dir := filepath.Join(home, ".config", "tynex", "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating sessions directory: %w", err)
	}
	return &SessionStore{sessionsDir: dir}, nil
}

// Save saves a session to disk.
func (s *SessionStore) Save(session *Session) error {
	session.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	path := filepath.Join(s.sessionsDir, session.ID+".json")
	return os.WriteFile(path, data, 0644)
}

// Load loads a session from disk.
func (s *SessionStore) Load(id string) (*Session, error) {
	path := filepath.Join(s.sessionsDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session %s: %w", id, err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session %s: %w", id, err)
	}
	return &session, nil
}

// List returns all saved sessions.
func (s *SessionStore) List() ([]Session, error) {
	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Session{}, nil
		}
		return nil, err
	}
	var sessions []Session
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			id := strings.TrimSuffix(entry.Name(), ".json")
			session, err := s.Load(id)
			if err != nil {
				continue
			}
			sessions = append(sessions, *session)
		}
	}
	return sessions, nil
}

// Delete removes a session from disk.
func (s *SessionStore) Delete(id string) error {
	path := filepath.Join(s.sessionsDir, id+".json")
	return os.Remove(path)
}

// Chat manages the interactive chat loop.
type Chat struct {
	agent            *ai.Agent
	cfg              *config.Config
	providerName     string
	fallbackProviders []ai.FallbackProvider
	store            *SessionStore
	sessionID        string
	reader           *bufio.Scanner
}

// NewChat creates a new interactive chat session.
func NewChat(cfg *config.Config, prov provider.Provider, providerName string, fallbackProviders []ai.FallbackProvider) (*Chat, error) {
	store, err := NewSessionStore()
	if err != nil {
		return nil, err
	}

	agent := ai.NewAgent(cfg, prov, "")
	// Set interactive confirmation for dangerous operations
	agent.ConfirmDangerous = ai.ConfirmInteractive
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	return &Chat{
		agent:             agent,
		cfg:               cfg,
		providerName:      providerName,
		fallbackProviders: fallbackProviders,
		store:             store,
		sessionID:         sessionID,
		reader:            bufio.NewScanner(os.Stdin),
	}, nil
}

// printBanner prints the welcome banner.
func (c *Chat) printBanner() {
	fmt.Println()
	fmt.Println(tui.RenderBanner(tui.BannerOptions{
		ShowLogo:    true,
		ShowDivider: true,
		Provider:    c.providerName,
		Model:       c.cfg.Model,
		Version:     "v0.1.0",
	}))
	fmt.Printf("Session: %s\n", c.sessionID)
	// Show fallback info if available
	if len(c.fallbackProviders) > 0 {
		var fbNames []string
		for _, fb := range c.fallbackProviders {
			fbNames = append(fbNames, fb.Name)
		}
		fmt.Printf("Fallbacks: %s\n", strings.Join(fbNames, ", "))
	}
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  /exit    — Exit the session")
	fmt.Println("  /clear   — Clear conversation history")
	fmt.Println("  /save    — Save session")
	fmt.Println("  /history — Show conversation history")
	fmt.Println("  /check   — Check configured providers")
	fmt.Println("  /help    — Show available commands")
	fmt.Println()
}

// Start begins the interactive chat loop.
func (c *Chat) Start(ctx context.Context) error {
	c.printBanner()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fmt.Print(tui.RenderPromptPrefix())
		if !c.reader.Scan() {
			break
		}

		input := strings.TrimSpace(c.reader.Text())
		if input == "" {
			continue
		}

		// Check for commands
		if strings.HasPrefix(input, "/") {
			if err := c.handleCommand(input); err != nil {
				return err
			}
			continue
		}

		// Send to AI (with fallback support)
		fmt.Println()
		fmt.Print(tui.RenderAssistantPrefix())

		// Accumulate response for saving
		var fullResponse strings.Builder
		streamFn := func(chunk string) {
			fmt.Print(chunk)
			fullResponse.WriteString(chunk)
		}

		resp, err := c.agent.SendMessageWithFallback(ctx, input, streamFn, c.fallbackProviders)
		if err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			continue
		}

		if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
			fmt.Printf("\n\n--- Tokens: %d in / %d out ---\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
		} else {
			fmt.Println()
		}

		// Show status bar with active model
		fmt.Println(tui.RenderStatusBar(c.cfg.Model))

		// Auto-save session
		c.autoSave()
	}

	return nil
}

// handleCommand processes a chat command.
func (c *Chat) handleCommand(cmd string) error {
	parts := strings.Fields(cmd)
	command := parts[0]

	switch command {
	case "/exit":
		fmt.Println("\nSessiya yakunlandi. Xayr!")
		return ErrExit

	case "/clear":
		c.agent.ClearHistory()
		fmt.Println("\n✅ Konversatsiya tarixi tozalandi.")

	case "/save":
		if err := c.saveSession(); err != nil {
			fmt.Printf("\n❌ Xatolik: %v\n", err)
		} else {
			fmt.Printf("\n✅ Sessiya saqlandi: %s\n", c.sessionID)
		}

	case "/history":
		messages := c.agent.GetHistory()
		if len(messages) == 0 {
			fmt.Println("\n📋 Konversatsiya bo'sh.")
			return nil
		}
		fmt.Printf("\n📋 Konversatsiya tarixi (%d xabar):\n", len(messages))
		for i, msg := range messages {
			role := "User"
			if msg.Role == "assistant" {
				role = "Tynex"
			}
			preview := msg.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Printf("  %d. [%s] %s\n", i+1, role, preview)
		}

	case "/help":
		fmt.Println()
		fmt.Println("Mavjud buyruqlar:")
		fmt.Println("  /exit    - Sessiyani yakunlash")
		fmt.Println("  /clear   - Konversatsiya tarixini tozalash")
		fmt.Println("  /save    - Sessiyani saqlash")
		fmt.Println("  /history - Konversatsiya tarixini ko'rish")
		fmt.Println("  /check   - Mavjud provayderlarni tekshirish")
		fmt.Println("  /help    - Yordam")
		fmt.Println()

	case "/check":
		fmt.Println("\n🔍 Provayderlarni tekshirish...")
		for name, pc := range c.cfg.Providers {
			status := "✅"
			key := c.cfg.APIKey(name)
			if key == "" {
				status = "❌"
			}
			fmt.Printf("  %s %s (base_url: %s, model: %s)\n", status, name, pc.BaseURL, pc.Model)
		}

	default:
		fmt.Printf("\nNoma'lum buyruq: %s. /help yozing.\n", command)
	}

	return nil
}

// saveSession saves the current session.
func (c *Chat) saveSession() error {
	session := &Session{
		ID:        c.sessionID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  c.agent.GetHistory(),
		Provider:  c.providerName,
		Model:     c.cfg.Model,
	}
	return c.store.Save(session)
}

// autoSave saves the session without printing messages.
func (c *Chat) autoSave() {
	_ = c.saveSession()
}

// ListSessions lists all saved sessions.
func (c *Chat) ListSessions() ([]Session, error) {
	return c.store.List()
}

// RunPrompt sends a single prompt and prints the response (for -p mode).
func (c *Chat) RunPrompt(ctx context.Context, prompt string) (*provider.Response, error) {
	resp, err := c.agent.SendMessageWithFallback(ctx, prompt, func(chunk string) {
		fmt.Print(chunk)
	}, c.fallbackProviders)
	if err != nil {
		return nil, err
	}

	if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		fmt.Printf("\n\n--- Tokens: %d in / %d out ---\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}

	// Show status bar with active model
	fmt.Println(tui.RenderStatusBar(c.cfg.Model))

	c.autoSave()
	return resp, nil
}

// LoadSession loads a session and restores the agent's history.
func (c *Chat) LoadSession(id string) error {
	session, err := c.store.Load(id)
	if err != nil {
		return err
	}
	c.sessionID = session.ID
	c.agent.ClearHistory()
	for _, msg := range session.Messages {
		c.agent.AddMessage(msg.Role, msg.Content)
	}
	return nil
}
