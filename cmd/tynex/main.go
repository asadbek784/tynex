package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/asadbek_784/tynex-cli/internal/ai"
	"github.com/asadbek_784/tynex-cli/internal/chat"
	"github.com/asadbek_784/tynex-cli/internal/config"
	"github.com/asadbek_784/tynex-cli/internal/provider"
	"github.com/asadbek_784/tynex-cli/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Check for -p flag (one-shot prompt)
	if len(os.Args) >= 3 && os.Args[1] == "-p" {
		prompt := strings.Join(os.Args[2:], " ")
		return runPrompt(prompt)
	}

	if len(os.Args) < 2 {
		// Default: start interactive chat
		return startChat()
	}

	command := os.Args[1]

	switch command {
	case "config":
		return handleConfig(os.Args[2:])
	case "init":
		return handleInit()
	case "provider":
		return handleProvider(os.Args[2:])
	case "session":
		return handleSession(os.Args[2:])
	case "use":
		return handleUse(os.Args[2:])
	case "chat":
		return startChat()
	case "help", "--help", "-h":
		printHelp()
		return nil
	case "version", "--version", "-v":
		printVersion()
		return nil
	default:
		// Treat unknown command as a prompt
		return runPrompt(command + " " + strings.Join(os.Args[2:], " "))
	}
}

// resolvedProviders holds the primary provider and any fallback providers.
type resolvedProviders struct {
	cfg              *config.Config
	primary          provider.Provider
	primaryName      string
	fallbackList     []ai.FallbackProvider
}

// runPrompt runs a one-shot prompt and exits.
func runPrompt(prompt string) error {
	rp, err := resolveWithFallbacks()
	if err != nil {
		return err
	}

	session, err := chat.NewChat(rp.cfg, rp.primary, rp.primaryName, rp.fallbackList)
	if err != nil {
		return fmt.Errorf("chat session yaratilmadi: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Println(tui.RenderBanner(tui.BannerOptions{
		ShowLogo:    false,
		ShowDivider: true,
		Provider:    rp.primaryName,
		Model:       rp.cfg.Model,
		Version:     "v0.1.0",
	}))

	fmt.Printf("%s%s\n\n", tui.RenderPromptPrefix(), prompt)
	fmt.Print(tui.RenderAssistantPrefix())

	_, err = session.RunPrompt(ctx, prompt)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	fmt.Println()
	return nil
}

// resolveWithFallbacks loads config and prepares the primary + fallback providers.
func resolveWithFallbacks() (*resolvedProviders, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config yuklanmadi: %w", err)
	}

	reg := provider.NewRegistry()

	// Collect all providers that have API keys
	type namedProvider struct {
		name string
		pCfg config.ProviderConfig
	}

	var providersWithKeys []namedProvider
	for name := range cfg.Providers {
		key := cfg.APIKey(name)
		if key != "" {
			providersWithKeys = append(providersWithKeys, namedProvider{name: name, pCfg: cfg.Providers[name]})
		}
	}

	if len(providersWithKeys) == 0 {
		return nil, fmt.Errorf("hech qanday API kalit topilmadi. 'tynex init' yoki 'TYNEX_<PROVIDER>_API_KEY' env o'zgaruvchisini o'rnating")
	}

	// The first matching provider is the primary.
	// Prioritize the default provider, otherwise use the first one with a key.
	primaryIdx := 0
	for i, np := range providersWithKeys {
		if np.name == cfg.DefaultProvider {
			primaryIdx = i
			break
		}
	}

	// Move primary to front
	providersWithKeys[0], providersWithKeys[primaryIdx] = providersWithKeys[primaryIdx], providersWithKeys[0]

	primary := providersWithKeys[0]
	primaryProv, err := reg.Get(primary.name, cfg.APIKey(primary.name), primary.pCfg.BaseURL, primary.pCfg.Model)
	if err != nil {
		return nil, fmt.Errorf("provayder yaratilmadi: %w", err)
	}

	// Build fallback list from remaining providers
	var fallbacks []ai.FallbackProvider
	for i := 1; i < len(providersWithKeys); i++ {
		np := providersWithKeys[i]
		prov, err := reg.Get(np.name, cfg.APIKey(np.name), np.pCfg.BaseURL, np.pCfg.Model)
		if err != nil {
			continue // Skip providers that fail to initialize
		}
		fallbacks = append(fallbacks, ai.FallbackProvider{
			Name:     np.name,
			Provider: prov,
			Model:    np.pCfg.Model,
		})
	}

	return &resolvedProviders{
		cfg:          cfg,
		primary:      primaryProv,
		primaryName:  primary.name,
		fallbackList: fallbacks,
	}, nil
}

// startChat loads config, creates provider, and starts interactive chat.
func startChat() error {
	rp, err := resolveWithFallbacks()
	if err != nil {
		return err
	}

	chatSession, err := chat.NewChat(rp.cfg, rp.primary, rp.primaryName, rp.fallbackList)
	if err != nil {
		return fmt.Errorf("chat session yaratilmadi: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := chatSession.Start(ctx); err != nil {
		if err == chat.ErrExit {
			return nil
		}
		return err
	}

	return nil
}

// handleConfig handles the "config" subcommand.
func handleConfig(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config yuklanmadi: %w", err)
	}

	if len(args) == 0 {
		// Show full config (same as "config list")
		fmt.Println(tui.RenderBanner(tui.BannerOptions{
			ShowLogo:    false,
			ShowDivider: true,
			Version:     "v0.1.0",
		}))
		fmt.Println(cfg.String())
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "set":
		return handleConfigSet(cfg, args[1:])
	case "path":
		path, err := config.ConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	case "add":
		return handleConfigAdd(cfg, args[1:])
	case "list":
		fmt.Println(tui.RenderBanner(tui.BannerOptions{
			ShowLogo:    false,
			ShowDivider: true,
			Version:     "v0.1.0",
		}))
		fmt.Println(cfg.String())
		return nil
	case "delete":
		return handleConfigDelete(cfg, args[1:])
	default:
		return fmt.Errorf("noma'lum config buyrug'i: %s (mavjud: set, add, list, delete, path)", subcommand)
	}
}

// handleConfigAdd adds a new provider configuration.
func handleConfigAdd(cfg *config.Config, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("ishlatilishi: tynex config add <name> <base_url> <model> [api_key]\n\nMisol: tynex config add openai https://api.openai.com/v1 gpt-4o sk-...")
	}

	name := strings.ToLower(args[0])
	baseURL := args[1]
	model := args[2]
	apiKey := ""
	if len(args) >= 4 {
		apiKey = args[3]
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	cfg.Providers[name] = config.ProviderConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}

	// If this is the first provider, set it as default
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = name
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("config saqlanmadi: %w", err)
	}
	fmt.Printf("✅ Provider '%s' qo'shildi (base_url: %s, model: %s)\n", name, baseURL, model)
	return nil
}

// handleConfigDelete removes a provider configuration.
func handleConfigDelete(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ishlatilishi: tynex config delete <name>\n\nMisol: tynex config delete openai")
	}

	name := strings.ToLower(args[0])
	if _, ok := cfg.Providers[name]; !ok {
		return fmt.Errorf("provider '%s' topilmadi", name)
	}

	delete(cfg.Providers, name)

	// If deleted provider was default, reset default
	if cfg.DefaultProvider == name {
		cfg.DefaultProvider = ""
		for n := range cfg.Providers {
			cfg.DefaultProvider = n
			break
		}
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("config saqlanmadi: %w", err)
	}
	fmt.Printf("✅ Provider '%s' o'chirildi\n", name)
	return nil
}

// handleConfigSet sets a configuration value.
func handleConfigSet(cfg *config.Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("ishlatilishi: tynex config set <key> <value>\n\nMisol: tynex config set default_provider anthropic")
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	switch key {
	case "default_provider":
		cfg.DefaultProvider = value
	case "model":
		cfg.Model = value
	case "max_tokens":
		fmt.Sscanf(value, "%d", &cfg.MaxTokens)
	case "temperature":
		fmt.Sscanf(value, "%f", &cfg.Temperature)
	case "system_prompt":
		cfg.SystemPrompt = value
	default:
		return fmt.Errorf("noma'lum config kaliti: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("config saqlanmadi: %w", err)
	}
	fmt.Printf("✅ %s = %s\n", key, value)
	return nil
}

// handleUse switches the active/default provider.
func handleUse(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ishlatilishi: tynex use <provider_name>\n\nMisol: tynex use anthropic")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config yuklanmadi: %w", err)
	}

	name := strings.ToLower(args[0])
	if _, ok := cfg.Providers[name]; !ok {
		return fmt.Errorf("provider '%s' topilmadi. Avval 'tynex config add %s <url> <model>' bilan qo'shing", name, name)
	}

	cfg.DefaultProvider = name
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("config saqlanmadi: %w", err)
	}

	p := cfg.Providers[name]
	fmt.Printf("✅ Aktiv provayder -> %s (base_url: %s, model: %s)\n", name, p.BaseURL, p.Model)
	return nil
}

// handleInit creates a default config and guides the user through setup.
func handleInit() error {
	// Check if config already exists
	path, err := config.ConfigPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		// Config exists, show it
		cfg, _ := config.Load()
		fmt.Println(tui.RenderBanner(tui.BannerOptions{
			ShowLogo:    true,
			ShowDivider: true,
			Version:     "v0.1.0",
		}))
		fmt.Println("✅ Konfiguratsiya fayli allaqachon mavjud:")
		fmt.Println(cfg.String())
		fmt.Println("Qayta sozlash uchun:")
		fmt.Println("  tynex config set default_provider <name>")
		fmt.Println("  tynex config add <name> <base_url> <model> [api_key]")
		return nil
	}

	cfg := config.DefaultConfig()

	fmt.Println(tui.RenderBanner(tui.BannerOptions{
		ShowLogo:    true,
		ShowDivider: true,
		Version:     "v0.1.0",
	}))

	// Ask for default provider
	fmt.Println("Qaysi AI provayderdan foydalanmoqchisiz?")
	fmt.Println("Mavjud: openai, anthropic, deepseek, groq, together")
	fmt.Println()
	fmt.Print("Provayder [openai]: ")
	providerName := readLine()
	if providerName == "" {
		providerName = "openai"
	}
	cfg.DefaultProvider = providerName

	// Ask for API key
	fmt.Print("API kalit: ")
	apiKey := readLine()
	if apiKey != "" {
		p := cfg.Providers[providerName]
		p.APIKey = apiKey
		cfg.Providers[providerName] = p
	}

	// Ask for base URL
	p := cfg.Providers[providerName]
	fmt.Printf("Base URL [%s]: ", p.BaseURL)
	baseURL := readLine()
	if baseURL != "" {
		p.BaseURL = baseURL
	}
	cfg.Providers[providerName] = p

	// Ask for model
	fmt.Printf("Model [%s]: ", p.Model)
	model := readLine()
	if model != "" {
		p.Model = model
	}
	cfg.Providers[providerName] = p

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("config saqlanmadi: %w", err)
	}

	fmt.Println()
	fmt.Printf("✅ Konfiguratsiya saqlandi: %s\n", path)
	fmt.Println()
	fmt.Println("Endi 'tynex' buyrug'i bilan ishga tushirishingiz mumkin!")
	fmt.Println()
	fmt.Println("💡 Maslahat: API kalitlarni .env faylida saqlash xavfsizroq:")
	fmt.Printf("   echo 'TYNEX_%s_API_KEY=sk-...' > .env\n", strings.ToUpper(providerName))

	return nil
}

// readLine reads a line from stdin using bufio.Scanner.
func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// handleProvider handles the "provider" subcommand.
func handleProvider(args []string) error {
	reg := provider.NewRegistry()

	if len(args) == 0 {
		fmt.Println("Mavjud provayderlar:")
		for _, name := range reg.AvailableProviders() {
			fmt.Printf("  - %s\n", name)
		}
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		fmt.Println("Mavjud provayderlar:")
		for _, name := range reg.AvailableProviders() {
			fmt.Printf("  - %s\n", name)
		}
		return nil
	default:
		return fmt.Errorf("noma'lum provider buyrug'i: %s (mavjud: list)", subcommand)
	}
}

// handleSession handles the "session" subcommand.
func handleSession(args []string) error {
	store, err := chat.NewSessionStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		sessions, err := store.List()
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("Hech qanday saqlangan sessiya yo'q.")
			return nil
		}
		fmt.Println("Saqlangan sessiyalar:")
		for _, s := range sessions {
			fmt.Printf("  - %s (%s, %d xabar)\n",
				s.ID, s.CreatedAt.Format("2006-01-02 15:04"), len(s.Messages))
		}
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		sessions, err := store.List()
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("Hech qanday saqlangan sessiya yo'q.")
			return nil
		}
		fmt.Println("Saqlangan sessiyalar:")
		for _, s := range sessions {
			fmt.Printf("  - %s (%s, %d xabar)\n",
				s.ID, s.CreatedAt.Format("2006-01-02 15:04"), len(s.Messages))
		}
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("ishlatilishi: tynex session delete <session_id>")
		}
		if err := store.Delete(args[1]); err != nil {
			return fmt.Errorf("sessiya o'chirilmadi: %w", err)
		}
		fmt.Printf("✅ Sessiya o'chirildi: %s\n", args[1])
	default:
		return fmt.Errorf("noma'lum session buyrug'i: %s (mavjud: list, delete)", subcommand)
	}

	return nil
}

// printHelp prints the help message.
func printHelp() {
	fmt.Println(tui.RenderBanner(tui.BannerOptions{
		ShowLogo:    true,
		ShowDivider: true,
		Version:     "v0.1.0",
	}))
	fmt.Println("ISHLATILISHI:")
	fmt.Println("  tynex                    Interaktiv chat sessiyasini boshlash")
	fmt.Println("  tynex chat              Interaktiv chat sessiyasini boshlash")
	fmt.Println("  tynex -p <prompt>       Bir martalik prompt bajarish")
	fmt.Println("  tynex <prompt>          To'g'ridan-to'g'ri prompt yozish")
	fmt.Println()
	fmt.Println("KONFIGURATSIYA:")
	fmt.Println("  tynex init              Boshlang'ich sozlash (interaktiv)")
	fmt.Println("  tynex config            Konfiguratsiyani ko'rish")
	fmt.Println("  tynex config list       Provayderlar ro'yxati")
	fmt.Println("  tynex config add <name> <url> <model> [api_key]")
	fmt.Println("  tynex config set <key> <value>")
	fmt.Println("  tynex config delete <name>")
	fmt.Println("  tynex config path       Konfiguratsiya fayli yo'li")
	fmt.Println()
	fmt.Println("PROVAYDERLAR:")
	fmt.Println("  tynex use <name>        Aktiv provayderni tanlash")
	fmt.Println("  tynex provider          Mavjud provayderlarni ko'rish")
	fmt.Println()
	fmt.Println("SESSIYALAR:")
	fmt.Println("  tynex session           Saqlangan sessiyalarni ko'rish")
	fmt.Println("  tynex session delete <id>")
	fmt.Println()
	fmt.Println("BOSHQA:")
	fmt.Println("  tynex help              Yordam")
	fmt.Println("  tynex version           Versiya")
	fmt.Println()
	fmt.Println("ATROF-MUHIT O'ZGARUVCHILARI:")
	fmt.Println("  TYNEX_DEFAULT_PROVIDER     Standart provayder")
	fmt.Println("  TYNEX_OPENAI_API_KEY       OpenAI API kaliti")
	fmt.Println("  TYNEX_ANTHROPIC_API_KEY    Anthropic API kaliti")
	fmt.Println("  TYNEX_DEEPSEEK_API_KEY     DeepSeek API kaliti")
	fmt.Println("  TYNEX_GROQ_API_KEY         Groq API kaliti")
	fmt.Println("  TYNEX_TOGETHER_API_KEY     Together AI API kaliti")
	fmt.Println("  TYNEX_MODEL                Model nomi")
	fmt.Println("  TYNEX_MAX_TOKENS           Maksimal tokenlar")
	fmt.Println("  TYNEX_TEMPERATURE          Temperatura (0.0 - 2.0)")
	fmt.Println()
	fmt.Println("KONFIGURATSIYA: ~/.config/tynex/config.yaml")
	fmt.Println("SESSIYALAR:     ~/.config/tynex/sessions/")
}

// printVersion prints the version.
func printVersion() {
	fmt.Println(tui.RenderBanner(tui.BannerOptions{
		ShowLogo:    true,
		ShowDivider: true,
		Version:     "v0.1.0",
	}))
}
