package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

func ExportRun(root, dest, sessionID, agentID, versionID, taskID string) error {
	record, err := GetRecord(root, sessionID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	if err := copyTree(recordFile(root, sessionID), filepath.Join(dest, "session-record.json"), map[string]bool{}); err != nil {
		return fmt.Errorf("copying session record: %w", err)
	}

	if record.FinalResponse != nil {
		if err := os.WriteFile(filepath.Join(dest, "response.md"), []byte(*record.FinalResponse), 0o644); err != nil {
			return fmt.Errorf("writing final response: %w", err)
		}
	}

	agentDest := filepath.Join(dest, "agent")
	if err := os.MkdirAll(agentDest, 0o755); err != nil {
		return err
	}
	if err := copyTree(agentFile(root, agentID), filepath.Join(agentDest, "agent.json"), map[string]bool{}); err != nil {
		return fmt.Errorf("copying agent metadata: %w", err)
	}
	if err := copyTree(versionFile(root, agentID, versionID), filepath.Join(agentDest, "version.json"), map[string]bool{}); err != nil {
		return fmt.Errorf("copying version metadata: %w", err)
	}
	if err := copyTree(versionSystemPromptFile(root, agentID, versionID), filepath.Join(agentDest, "version.system.md"), map[string]bool{}); err != nil {
		return fmt.Errorf("copying system prompt: %w", err)
	}

	skillsSrc := versionSkillsDir(root, agentID, versionID)
	if _, err := os.Stat(skillsSrc); err == nil {
		if err := copyTree(skillsSrc, filepath.Join(agentDest, "skills"), map[string]bool{}); err != nil {
			return fmt.Errorf("copying skills: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	taskDest := filepath.Join(dest, "task")
	if err := os.MkdirAll(taskDest, 0o755); err != nil {
		return err
	}
	if err := copyTree(taskFile(root, taskID), filepath.Join(taskDest, "task.json"), map[string]bool{}); err != nil {
		return fmt.Errorf("copying task metadata: %w", err)
	}
	if err := copyTree(taskPromptFile(root, taskID), filepath.Join(taskDest, "task.prompt.md"), map[string]bool{}); err != nil {
		return fmt.Errorf("copying task prompt: %w", err)
	}

	return nil
}
