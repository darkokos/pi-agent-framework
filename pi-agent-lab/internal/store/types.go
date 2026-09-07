package store

type Agent struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	CreatedAt   int64    `json:"createdAt"`
	Versions    []string `json:"versions"`
}

type SystemPromptRef struct {
	Mode string `json:"mode"` // "replace" | "append"
	File string `json:"file"`
}

type ToolsConfig struct {
	Mode  string   `json:"mode"` // "all" | "none" | "builtin-only" | "allowlist" | "denylist"
	Names []string `json:"names,omitempty"`
}

type ResourceIsolation struct {
	Isolate bool     `json:"isolate"`
	Paths   []string `json:"paths,omitempty"`
}

type AgentVersion struct {
	AgentID      string            `json:"agentId"`
	VersionID    string            `json:"versionId"`
	CreatedAt    int64             `json:"createdAt"`
	Model        string            `json:"model"`
	Provider     string            `json:"provider,omitempty"`
	Thinking     string            `json:"thinking,omitempty"`
	SystemPrompt SystemPromptRef   `json:"systemPrompt"`
	Tools        ToolsConfig       `json:"tools"`
	Extensions   ResourceIsolation `json:"extensions"`
	Skills       ResourceIsolation `json:"skills"`
	Notes        string            `json:"notes,omitempty"`
}

type Task struct {
	ID         string `json:"id"`
	PromptFile string `json:"promptFile"`
	Notes      string `json:"notes,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
}

type AgentVersionInput struct {
	SystemPromptFile string
	Model            string
	Provider         string
	Thinking         string
	Notes            string
	Tools            ToolsConfig
	Extensions       ResourceIsolation
	Skills           ResourceIsolation
}
