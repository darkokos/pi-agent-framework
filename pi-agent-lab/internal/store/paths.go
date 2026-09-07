package store

import (
	"os"
	"path/filepath"
)

func root() (string, error) {
	return os.Getwd()
}

func agentsDir() (string, error) {
	r, err := root()
	if err != nil {
		return "", err
	}

	return filepath.Join(r, "agents"), nil
}

func tasksDir() (string, error) {
	r, err := root()
	if err != nil {
		return "", err
	}

	return filepath.Join(r, "tasks"), nil
}

func agentDir(agentID string) (string, error) {
	d, err := agentsDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(d, agentID), nil
}

func agentFile(agentID string) (string, error) {
	d, err := agentDir(agentID)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, "agent.json"), nil
}

func versionsDir(agentID string) (string, error) {
	d, err := agentDir(agentID)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, "versions"), nil
}

func versionFile(agentID, versionID string) (string, error) {
	d, err := versionsDir(agentID)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, versionID+".json"), nil
}

func versionSystemPromptFile(agentID, versionID string) (string, error) {
	d, err := versionsDir(agentID)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, versionID+".system.md"), nil
}

func taskFile(taskID string) (string, error) {
	d, err := tasksDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(d, taskID+".json"), nil
}

func taskPromptFile(taskID string) (string, error) {
	d, err := tasksDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(d, taskID+".prompt.md"), nil
}
