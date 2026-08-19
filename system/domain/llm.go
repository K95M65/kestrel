package domain

import "strings"

// LLMAuthKind is how the operator hands this provider credentials.
type LLMAuthKind string

const (
	LLMAuthAPIKey     LLMAuthKind = "api_key"
	LLMAuthDeviceCode LLMAuthKind = "device_code"
	LLMAuthBYO        LLMAuthKind = "byo"
)

// LLMProviderField is an extra form field (Cloudflare account id, etc.).
type LLMProviderField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

type LLMProvider struct {
	Key          string             `json:"key"`
	Name         string             `json:"name"`
	Auth         LLMAuthKind        `json:"auth"`
	BaseURL      string             `json:"base_url,omitempty"`
	DefaultModel string             `json:"default_model,omitempty"`
	DocsURL      string             `json:"docs_url,omitempty"`
	Hint         string             `json:"hint,omitempty"`
	Fields       []LLMProviderField `json:"fields,omitempty"`
	OpenClawAPI  string             `json:"openclaw_api,omitempty"`
}

// ListProviders is the OpenCode-/connect/ catalog Autonomous setup can offer.
// Device-code login is implemented for xAI (SuperGrok). Everyone else is
// API key or a local OpenAI-compatible URL; extra Fields fill {placeholders}
// in BaseURL.
var ListProviders = []LLMProvider{
	{Key: "xai", Name: "xAI Grok (SuperGrok)", Auth: LLMAuthDeviceCode, BaseURL: "https://api.x.ai/v1", DefaultModel: "grok-4.6", DocsURL: "https://opencode.ai/docs/providers/#xai", Hint: "Sign in with your Grok / SuperGrok account (subscription, not a console xai- key). You can still paste a console key instead."},
	{Key: "kimi", Name: "Kimi (Moonshot)", Auth: LLMAuthAPIKey, BaseURL: "https://api.moonshot.ai/v1", DefaultModel: "kimi-k2.5", DocsURL: "https://platform.moonshot.ai/console", Hint: "Moonshot AI API key from platform.moonshot.ai."},
	{Key: "kimi-cn", Name: "Kimi (Moonshot China)", Auth: LLMAuthAPIKey, BaseURL: "https://api.moonshot.cn/v1", DefaultModel: "kimi-k2.5", DocsURL: "https://platform.moonshot.cn/console"},
	{Key: "cloudflare-workers-ai", Name: "Cloudflare Workers AI", Auth: LLMAuthAPIKey, BaseURL: "https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1", DefaultModel: "@cf/moonshotai/kimi-k2.5", DocsURL: "https://opencode.ai/docs/providers/#cloudflare-workers-ai", Hint: "Workers AI REST API token plus Account ID. Hosted Kimi lives here.", Fields: []LLMProviderField{{Key: "account_id", Label: "Account ID", Placeholder: "32-character account id"}}},
	{Key: "cloudflare-ai-gateway", Name: "Cloudflare AI Gateway", Auth: LLMAuthAPIKey, BaseURL: "https://gateway.ai.cloudflare.com/v1/{account_id}/{gateway_id}", DefaultModel: "@cf/moonshotai/kimi-k2.5", DocsURL: "https://opencode.ai/docs/providers/#cloudflare-ai-gateway", Fields: []LLMProviderField{{Key: "account_id", Label: "Account ID"}, {Key: "gateway_id", Label: "Gateway ID"}}},
	{Key: "openai", Name: "OpenAI", Auth: LLMAuthAPIKey, BaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-4.1", DocsURL: "https://platform.openai.com/api-keys"},
	{Key: "anthropic", Name: "Anthropic", Auth: LLMAuthAPIKey, BaseURL: "https://api.anthropic.com", DefaultModel: "claude-sonnet-4-5", OpenClawAPI: "anthropic-messages", DocsURL: "https://console.anthropic.com/", Hint: "API key. Claude Pro/Max browser login is not on the device yet."},
	{Key: "google", Name: "Google Gemini API", Auth: LLMAuthAPIKey, BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", DefaultModel: "gemini-2.5-flash"},
	{Key: "openrouter", Name: "OpenRouter", Auth: LLMAuthAPIKey, BaseURL: "https://openrouter.ai/api/v1", DefaultModel: "openrouter/auto"},
	{Key: "groq", Name: "Groq", Auth: LLMAuthAPIKey, BaseURL: "https://api.groq.com/openai/v1", DefaultModel: "llama-3.3-70b-versatile"},
	{Key: "cerebras", Name: "Cerebras", Auth: LLMAuthAPIKey, BaseURL: "https://api.cerebras.ai/v1", DefaultModel: "qwen-3-coder-480b"},
	{Key: "mistral", Name: "Mistral AI", Auth: LLMAuthAPIKey, BaseURL: "https://api.mistral.ai/v1", DefaultModel: "mistral-large-latest"},
	{Key: "deepseek", Name: "DeepSeek", Auth: LLMAuthAPIKey, BaseURL: "https://api.deepseek.com", DefaultModel: "deepseek-chat", DocsURL: "https://platform.deepseek.com/"},
	{Key: "together", Name: "Together AI", Auth: LLMAuthAPIKey, BaseURL: "https://api.together.xyz/v1", DefaultModel: "moonshotai/Kimi-K2-Instruct"},
	{Key: "fireworks", Name: "Fireworks AI", Auth: LLMAuthAPIKey, BaseURL: "https://api.fireworks.ai/inference/v1", DefaultModel: "accounts/fireworks/models/kimi-k2-instruct"},
	{Key: "huggingface", Name: "Hugging Face", Auth: LLMAuthAPIKey, BaseURL: "https://router.huggingface.co/v1", DefaultModel: "moonshotai/Kimi-K2-Instruct"},
	{Key: "minimax", Name: "MiniMax", Auth: LLMAuthAPIKey, BaseURL: "https://api.minimax.io/v1", DefaultModel: "MiniMax-M2.1"},
	{Key: "nvidia", Name: "NVIDIA", Auth: LLMAuthAPIKey, BaseURL: "https://integrate.api.nvidia.com/v1", DefaultModel: "moonshotai/kimi-k2-instruct"},
	{Key: "zai", Name: "Z.AI (GLM)", Auth: LLMAuthAPIKey, BaseURL: "https://api.z.ai/api/paas/v4", DefaultModel: "glm-4.6"},
	{Key: "vercel-ai-gateway", Name: "Vercel AI Gateway", Auth: LLMAuthAPIKey, BaseURL: "https://ai-gateway.vercel.sh/v1", DefaultModel: "anthropic/claude-sonnet-4"},
	{Key: "opencode", Name: "OpenCode Zen", Auth: LLMAuthAPIKey, BaseURL: "https://opencode.ai/zen/v1", DefaultModel: "gpt-5-nano", DocsURL: "https://opencode.ai/auth"},
	{Key: "opencode-go", Name: "OpenCode Go", Auth: LLMAuthAPIKey, BaseURL: "https://opencode.ai/zen/v1", DefaultModel: "gpt-5-nano", DocsURL: "https://opencode.ai/auth"},
	{Key: "github-copilot", Name: "GitHub Copilot", Auth: LLMAuthAPIKey, BaseURL: "https://api.githubcopilot.com", DefaultModel: "gpt-4.1", Hint: "OpenCode uses GitHub device login. Paste a Copilot token for now."},
	{Key: "openai-codex", Name: "OpenAI Codex", Auth: LLMAuthAPIKey, BaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-5.1-codex"},
	{Key: "google-vertex", Name: "Google Vertex AI", Auth: LLMAuthAPIKey, BaseURL: "", Hint: "Set GOOGLE_CLOUD_PROJECT on the device; paste a key or ADC path in API key."},
	{Key: "google-antigravity", Name: "Google Antigravity", Auth: LLMAuthAPIKey, BaseURL: "", Hint: "OAuth from OpenCode /connect. Paste a token until device login lands."},
	{Key: "google-gemini-cli", Name: "Google Gemini CLI", Auth: LLMAuthAPIKey, BaseURL: "", Hint: "OAuth from Gemini CLI. Paste a token until device login lands."},
	{Key: "ollama", Name: "Ollama (local)", Auth: LLMAuthBYO, BaseURL: "http://127.0.0.1:11434/v1", DefaultModel: "llama3.2", Hint: "No cloud key. API key can be the dummy value ollama. Point Base URL at a LAN box if the model is not on the robot."},
	{Key: "lmstudio", Name: "LM Studio (local)", Auth: LLMAuthBYO, BaseURL: "http://127.0.0.1:1234/v1", DefaultModel: "local", Hint: "Dummy API key lmstudio is fine."},
	{Key: "custom", Name: "Custom (OpenAI-compatible)", Auth: LLMAuthBYO, BaseURL: "", Hint: "Any OpenAI-compatible /v1 endpoint."},
}

// LookupLLMProvider returns a catalog entry by key.
func LookupLLMProvider(key string) (LLMProvider, bool) {
	key = strings.TrimSpace(strings.ToLower(key))
	for _, p := range ListProviders {
		if p.Key == key {
			return p, true
		}
	}
	return LLMProvider{}, false
}

// ExpandProviderBaseURL substitutes {field} placeholders from extras.
func ExpandProviderBaseURL(base string, extras map[string]string) string {
	out := base
	for k, v := range extras {
		out = strings.ReplaceAll(out, "{"+k+"}", strings.TrimSpace(v))
	}
	return out
}

type LLMModelCapabilities struct {
	SupportsReasoning       bool `json:"supportsReasoning"`
	SupportsVision          bool `json:"supportsVision"`
	SupportsFunctionCalling bool `json:"supportsFunctionCalling"`
}

type LLMModel struct {
	Key           string                `json:"key"`
	Name          string                `json:"name"`
	Reasoning     bool                  `json:"reasoning"`
	Input         []string              `json:"input"`
	ContextWindow *int                  `json:"contextWindow"`
	MaxTokens     *int                  `json:"maxTokens"`
	Privacy       string                `json:"privacy"`
	Capabilities  *LLMModelCapabilities `json:"capabilities"`
}

// OpenClawAPIType returns the OpenClaw provider api type from raw substring check on Key and Name.
// e.g. "claude" -> "anthropic-messages", "gpt" -> "openai-completions", unknown -> "openai-completions".
func (m LLMModel) OpenClawAPIType() string {
	raw := strings.ToLower(m.Key + " " + m.Name)
	if strings.Contains(raw, "claude") {
		return "anthropic-messages"
	}
	if strings.Contains(raw, "gpt") {
		return "openai-completions"
	}
	return "openai-completions"
}

type LLMModelsListResponse struct {
	Count int `json:"count"`
	// Version is the upstream catalog version. The set-default-model flow only
	// applies default_model / default_image_model when this is greater than the
	// device's persisted DefaultModelVersion (avoids redundant gateway restarts).
	Version int `json:"version"`
	// DefaultModel is the upstream-recommended primary text model key.
	DefaultModel string `json:"default_model"`
	// DefaultImageModel is the upstream-recommended vision/image model key.
	DefaultImageModel string `json:"default_image_model"`
	// API is the wire protocol the autonomous provider speaks (e.g.
	// "anthropic-messages"). Written into models.providers.autonomous.api at
	// setup and overwritten on each sync. Empty falls back to the built-in
	// default (autonomousProviderAPI).
	API    string     `json:"api"`
	Models []LLMModel `json:"models"`
}
