package registry

import "os"

func Root() (string, error) {
	if v := os.Getenv("PI_AGENT_REGISTRY_DIR"); v != "" {
		return v, nil
	}

	return os.Getwd()
}
