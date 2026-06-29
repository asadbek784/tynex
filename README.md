# Tynex — Provider-Agnostic AI CLI Coding Agent

**Tynex** — bu Go tilida yozilgan CLI (Command-Line Interface) agent bo'lib, Claude Code / Antigravity CLI'ga o'xshash ishlaydi, lekin asosiy farqi: **istalgan AI provayderning API kalitini qo'shib ishlatish imkoniyati**.

## ✨ Xususiyatlar

- **Provider-agnostic** — OpenAI, Anthropic, DeepSeek, Groq, Together AI va boshqa OpenAI-compatible API'lar bilan ishlaydi
- **Interaktiv chat** — terminal ichida to'g'ridan-to'g'ri AI bilan muloqot
- **Tool-calling** — fayl o'qish/yozish, kod qidirish, terminal buyruqlarini bajarish
- **Gradient ASCII logo** — chiroyli, rangli TYNEX logotipi (NO_COLOR qo'llab-quvvatlanadi)
- **Fallback tizimi** — asosiy provayder ishlamasa, boshqa provayderga avtomatik o'tish
- **Sessiyalarni saqlash** — chat tarixini saqlash va qayta yuklash
- **Xavfsizlik** — fayl yozish, o'chirish va shell buyruqlaridan oldin tasdiq so'rash
- **Streaming javoblar** — token-by-token real-time javob

## 🚀 O'rnatish

```bash
# 1. Go'ni o'rnating (agar o'rnatilmagan bo'lsa)
# https://go.dev/dl/

# 2. Reponi klonlang
git clone https://github.com/asadbek784/tynex.git
cd tynex

# 3. Binarni yarating
go build -o tynex ./cmd/tynex

# (ixtiyoriy) PATH'ga qo'shing
sudo mv tynex /usr/local/bin/
```

## 🎯 Ishlatish

### Boshlang'ich sozlash

```bash
tynex init
```

### Interaktiv chat

```bash
tynex
# yoki
tynex chat
```

### Bir martalik prompt

```bash
tynex -p "Hello, world!"
tynex "src/ papkasidagi barcha .go fayllarini sanab ber"
```

### Provayderlarni boshqarish

```bash
# Provayder qo'shish
tynex config add openai https://api.openai.com/v1 gpt-4o sk-...

# Aktiv provayderni tanlash
tynex use anthropic

# Barcha provayderlarni ko'rish
tynex config list

# Provayder o'chirish
tynex config delete groq
```

### Sessiyalar

```bash
tynex session          # Barcha sessiyalarni ko'rish
tynex session delete <id>
```

### Yordam

```bash
tynex help
tynex version
```

## ⚙️ Konfiguratsiya

Konfiguratsiya fayli: `~/.config/tynex/config.yaml`

```yaml
default_provider: openai
model: gpt-4o
max_tokens: 4096
temperature: 0.7
system_prompt: "You are Tynex, an AI-powered CLI coding assistant."
providers:
  openai:
    base_url: https://api.openai.com/v1
    model: gpt-4o
  anthropic:
    base_url: https://api.anthropic.com/v1
    model: claude-sonnet-4-20250514
```

API kalitlarni `.env` fayli orqali ham sozlash mumkin:

```bash
echo "TYNEX_OPENAI_API_KEY=sk-..." > .env
echo "TYNEX_ANTHROPIC_API_KEY=sk-ant-..." >> .env
```

Yoki atrof-muhit o'zgaruvchilari orqali:

```bash
export TYNEX_OPENAI_API_KEY=sk-...
export TYNEX_DEFAULT_PROVIDER=anthropic
```

### Mavjud atrof-muhit o'zgaruvchilari

| O'zgaruvchi | Tavsif |
|---|---|
| `TYNEX_DEFAULT_PROVIDER` | Standart provayder |
| `TYNEX_MODEL` | Model nomi |
| `TYNEX_MAX_TOKENS` | Maksimal tokenlar |
| `TYNEX_TEMPERATURE` | Temperatura (0.0 - 2.0) |
| `TYNEX_<NAME>_API_KEY` | Provayder API kaliti |
| `TYNEX_<NAME>_BASE_URL` | Provayder base URL |
| `TYNEX_<NAME>_MODEL` | Provayder modeli |

## 🖥️ Buyruqlar

```
tynex                    Interaktiv chat
tynex chat               Interaktiv chat
tynex -p <prompt>        Bir martalik prompt
tynex <prompt>           To'g'ridan-to'g'ri prompt
tynex init               Boshlang'ich sozlash
tynex config             Konfiguratsiyani ko'rish
tynex config list        Provayderlar ro'yxati
tynex config add ...     Provayder qo'shish
tynex config set ...     Sozlamalarni o'zgartirish
tynex config delete ...  Provayder o'chirish
tynex config path        Config fayli yo'li
tynex use <name>         Aktiv provayder
tynex provider           Provayderlar ro'yxati
tynex session            Sessiyalar
tynex help               Yordam
tynex version            Versiya
```

### Chat ichidagi buyruqlar

```
/exit    — Sessiyani yakunlash
/clear   — Konversatsiya tarixini tozalash
/save    — Sessiyani saqlash
/history — Konversatsiya tarixini ko'rish
/check   — Provayderlarni tekshirish
/help    — Yordam
```

## 🛠️ Texnologiyalar

- **Go 1.22+** — asosiy til
- **Lipgloss** — gradient logo va TUI ranglari
- **YAML** — konfiguratsiya fayllari
- **godotenv** — .env fayllarni o'qish

## 📁 Loyiha tuzilishi

```
tynex/
├── cmd/tynex/main.go          — CLI entry point
├── internal/
│   ├── ai/agent.go            — Agent loop, tool-calling, fallback
│   ├── chat/chat.go           — Interaktiv chat, sessiyalar
│   ├── config/config.go       — YAML + .env + config
│   ├── provider/              — Provider interface, OpenAI-compatible
│   ├── tui/logo.go            — Gradient ASCII logo + TUI
│   ├── fileops/fileops.go     — Fayl operatsiyalari
│   └── shell/shell.go         — Shell buyruqlari
├── .gitignore
├── README.md
├── LICENSE
├── knowledge.md
└── go.mod
```

## ⚠️ Xavfsizlik

- API kalitlar **hech qachon** log, kod yoki commit'ga yozilmaydi
- Fayl yozish, o'chirish va shell buyruqlari **har doim** tasdiq talab qiladi
- `.gitignore` maxfiy fayllarni (`.env`, `config.yaml`, `*.key`) himoyalaydi

## 📄 Litsenziya

MIT License — qarang: [LICENSE](LICENSE)

## 🔗 Havolalar

- [GitHub](https://github.com/asadbek784/tynex)
- [Go Documentation](https://go.dev/doc/)
