package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func CreateAgent(id, name, description string) (*Agent, error) {
	file, err := agentFile(id)
	if err != nil {
		return nil, err
	}

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

	dir, err := agentDir(id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := writeJSON(file, agent); err != nil {
		return nil, err
	}

	return agent, nil
}

func CreateAgentVersion(agentID, versionID string, in AgentVersionInput) (*AgentVersion, error) {
	aFile, err := agentFile(agentID)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := readJSON(aFile, &agent); err != nil {
		return nil, fmt.Errorf("agent %q not found: %w", agentID, err)
	}

	vFile, err := versionFile(agentID, versionID)
	if err != nil {
		return nil, err
	}
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

	vDir, err := versionsDir(agentID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(vDir, 0o755); err != nil {
		return nil, err
	}

	promptFile, err := versionSystemPromptFile(agentID, versionID)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(promptFile, promptContent, 0o644); err != nil {
		return nil, err
	}
	if err := writeJSON(vFile, version); err != nil {
		return nil, err
	}

	agent.Versions = append(agent.Versions, versionID)
	if err := writeJSON(aFile, &agent); err != nil {
		return nil, err
	}

	return version, nil
}

func CreateTask(id, promptFile, notes string) (*Task, error) {
	file, err := taskFile(id)
	if err != nil {
		return nil, err
	}

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

	dir, err := tasksDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	pFile, err := taskPromptFile(id)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(pFile, promptContent, 0o644); err != nil {
		return nil, err
	}
	if err := writeJSON(file, task); err != nil {
		return nil, err
	}

	return task, nil
}

func ListAgents() ([]*Agent, error) {
	dir, err := agentsDir()
	if err != nil {
		return nil, err
	}

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

		file, err := agentFile(entry.Name())
		if err != nil {
			return nil, err
		}

		var agent Agent
		if err := readJSON(file, &agent); err != nil {
			continue
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

func ListTasks() ([]*Task, error) {
	dir, err := tasksDir()
	if err != nil {
		return nil, err
	}

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
