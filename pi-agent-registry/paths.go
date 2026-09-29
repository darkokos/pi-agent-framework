package registry

import "path/filepath"

func agentsDir(root string) string { return filepath.Join(root, "agents") }

func tasksDir(root string) string { return filepath.Join(root, "tasks") }

func agentDir(root, agentID string) string {
	return filepath.Join(agentsDir(root), agentID)
}

func agentFile(root, agentID string) string {
	return filepath.Join(agentDir(root, agentID), "agent.json")
}

func versionsDir(root, agentID string) string {
	return filepath.Join(agentDir(root, agentID), "versions")
}

func versionFile(root, agentID, versionID string) string {
	return filepath.Join(versionsDir(root, agentID), versionID+".json")
}

func versionSystemPromptFile(root, agentID, versionID string) string {
	return filepath.Join(versionsDir(root, agentID), versionID+".system.md")
}

func versionSkillsDir(root, agentID, versionID string) string {
	return filepath.Join(versionsDir(root, agentID), versionID+".skills")
}

func taskFile(root, taskID string) string {
	return filepath.Join(tasksDir(root), taskID+".json")
}

func taskPromptFile(root, taskID string) string {
	return filepath.Join(tasksDir(root), taskID+".prompt.md")
}
