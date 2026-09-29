package registry

type Agent struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Versions    []string `json:"versions"`
}

type SystemPromptRef struct {
	Mode string `json:"mode"` // "replace" | "append"
	File string `json:"file"`
}

type ToolsConfig struct {
	Mode  string   `json:"mode"` // "all" | "none" | "extensions-only" | "allowlist" | "denylist"
	Names []string `json:"names,omitempty"`
}

type ExtensionsConfig struct {
	Isolate bool     `json:"isolate"`
	Sources []string `json:"sources,omitempty"`
}

type SkillsConfig struct {
	Isolate bool     `json:"isolate"`
	Names   []string `json:"names,omitempty"`
}

type AgentVersion struct {
	AgentID      string           `json:"agentId"`
	VersionID    string           `json:"versionId"`
	Model        string           `json:"model"`
	Provider     string           `json:"provider,omitempty"`
	Thinking     string           `json:"thinking,omitempty"`
	SystemPrompt SystemPromptRef  `json:"systemPrompt"`
	Tools        ToolsConfig      `json:"tools"`
	Extensions   ExtensionsConfig `json:"extensions"`
	Skills       SkillsConfig     `json:"skills"`
	Notes        string           `json:"notes,omitempty"`
}

type Task struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	PromptFile string `json:"promptFile"`
	Notes      string `json:"notes,omitempty"`
}
