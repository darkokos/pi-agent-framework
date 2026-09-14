package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func GetAgent(root, agentID string) (*Agent, error) {
	var agent Agent
	if err := readJSON(agentFile(root, agentID), &agent); err != nil {
		return nil, fmt.Errorf("agent %q not found: %w", agentID, err)
	}

	return &agent, nil
}

func GetAgentVersion(root, agentID, versionID string) (*AgentVersion, error) {
	var version AgentVersion
	if err := readJSON(versionFile(root, agentID, versionID), &version); err != nil {
		return nil, fmt.Errorf("version %q of agent %q not found: %w", versionID, agentID, err)
	}

	return &version, nil
}

func GetTask(root, taskID string) (*Task, error) {
	var task Task
	if err := readJSON(taskFile(root, taskID), &task); err != nil {
		return nil, fmt.Errorf("task %q not found: %w", taskID, err)
	}

	return &task, nil
}

func ReadVersionSystemPrompt(root, agentID, versionID string) (string, error) {
	content, err := os.ReadFile(versionSystemPromptFile(root, agentID, versionID))
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func TaskPromptPath(root, taskID string) string {
	return taskPromptFile(root, taskID)
}

func CreateAgent(root, id, name, description string) (*Agent, error) {
	file := agentFile(root, id)
	if _, err := os.Stat(file); err == nil {
		return nil, fmt.Errorf("agent %q already exists", id)
	}

	agent := &Agent{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UnixMilli(),
		Versions:    []string{},
	}

	if err := os.MkdirAll(agentDir(root, id), 0o755); err != nil {
		return nil, err
	}
	if err := writeJSON(file, agent); err != nil {
		return nil, err
	}

	return agent, nil
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

func CreateAgentVersion(root, agentID, versionID string, in AgentVersionInput) (*AgentVersion, error) {
	agent, err := GetAgent(root, agentID)
	if err != nil {
		return nil, err
	}

	vFile := versionFile(root, agentID, versionID)
	if _, err := os.Stat(vFile); err == nil {
		return nil, fmt.Errorf("version %q already exists for agent %q", versionID, agentID)
	}

	promptContent, err := os.ReadFile(in.SystemPromptFile)
	if err != nil {
		return nil, fmt.Errorf("reading system prompt file: %w", err)
	}

	version := &AgentVersion{
		AgentID:   agentID,
		VersionID: versionID,
		CreatedAt: time.Now().UnixMilli(),
		Model:     in.Model,
		Provider:  in.Provider,
		Thinking:  in.Thinking,
		SystemPrompt: SystemPromptRef{
			Mode: "replace",
			File: versionID + ".system.md",
		},
		Tools:      in.Tools,
		Extensions: in.Extensions,
		Skills:     in.Skills,
		Notes:      in.Notes,
	}

	if err := os.MkdirAll(versionsDir(root, agentID), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(versionSystemPromptFile(root, agentID, versionID), promptContent, 0o644); err != nil {
		return nil, err
	}
	if err := writeJSON(vFile, version); err != nil {
		return nil, err
	}

	agent.Versions = append(agent.Versions, versionID)
	if err := writeJSON(agentFile(root, agentID), agent); err != nil {
		return nil, err
	}

	return version, nil
}

func CreateTask(root, id, promptFile, notes string) (*Task, error) {
	file := taskFile(root, id)
	if _, err := os.Stat(file); err == nil {
		return nil, fmt.Errorf("task %q already exists", id)
	}

	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w", err)
	}

	task := &Task{
		ID:         id,
		PromptFile: id + ".prompt.md",
		Notes:      notes,
		CreatedAt:  time.Now().UnixMilli(),
	}

	if err := os.MkdirAll(tasksDir(root), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(taskPromptFile(root, id), promptContent, 0o644); err != nil {
		return nil, err
	}
	if err := writeJSON(file, task); err != nil {
		return nil, err
	}

	return task, nil
}

func ReplaceAgent(root string, agent *Agent) (*Agent, error) {
	if _, err := GetAgent(root, agent.ID); err != nil {
		return nil, err
	}

	if err := writeJSON(agentFile(root, agent.ID), agent); err != nil {
		return nil, err
	}

	return agent, nil
}

func ReplaceAgentVersion(root, agentID, versionID string, version *AgentVersion, newSystemPromptFile string) (*AgentVersion, error) {
	if _, err := GetAgentVersion(root, agentID, versionID); err != nil {
		return nil, err
	}

	if newSystemPromptFile != "" {
		content, err := os.ReadFile(newSystemPromptFile)
		if err != nil {
			return nil, fmt.Errorf("reading system prompt file: %w", err)
		}

		if err := os.WriteFile(versionSystemPromptFile(root, agentID, versionID), content, 0o644); err != nil {
			return nil, err
		}
	}

	if err := writeJSON(versionFile(root, agentID, versionID), version); err != nil {
		return nil, err
	}

	return version, nil
}

func ReplaceTask(root string, task *Task, newPromptFile string) (*Task, error) {
	if _, err := GetTask(root, task.ID); err != nil {
		return nil, err
	}

	if newPromptFile != "" {
		content, err := os.ReadFile(newPromptFile)
		if err != nil {
			return nil, fmt.Errorf("reading prompt file: %w", err)
		}

		if err := os.WriteFile(taskPromptFile(root, task.ID), content, 0o644); err != nil {
			return nil, err
		}
	}

	if err := writeJSON(taskFile(root, task.ID), task); err != nil {
		return nil, err
	}

	return task, nil
}

func DeleteAgent(root, agentID string) error {
	if _, err := GetAgent(root, agentID); err != nil {
		return err
	}

	return os.RemoveAll(agentDir(root, agentID))
}

func DeleteAgentVersion(root, agentID, versionID string) error {
	agent, err := GetAgent(root, agentID)
	if err != nil {
		return err
	}
	if _, err := GetAgentVersion(root, agentID, versionID); err != nil {
		return err
	}

	if err := os.Remove(versionFile(root, agentID, versionID)); err != nil {
		return err
	}
	if err := os.Remove(versionSystemPromptFile(root, agentID, versionID)); err != nil {
		return err
	}

	agent.Versions = slices.DeleteFunc(agent.Versions, func(v string) bool { return v == versionID })

	return writeJSON(agentFile(root, agentID), agent)
}

func DeleteTask(root, taskID string) error {
	if _, err := GetTask(root, taskID); err != nil {
		return err
	}

	if err := os.Remove(taskFile(root, taskID)); err != nil {
		return err
	}

	return os.Remove(taskPromptFile(root, taskID))
}

func ListAgents(root string) ([]*Agent, error) {
	dir := agentsDir(root)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var agents []*Agent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		var agent Agent
		if err := readJSON(agentFile(root, entry.Name()), &agent); err != nil {
			continue
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

func ListTasks(root string) ([]*Task, error) {
	dir := tasksDir(root)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var task Task
		if err := readJSON(filepath.Join(dir, entry.Name()), &task); err != nil {
			continue
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}
