package edgeagent

import "os"

func readVersionFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeVersionToFile(filePath, version string) error {
	return os.WriteFile(filePath, []byte(version), 0o600)
}
