package config

import (
	"bufio"
	"os"
	"strings"
)

func resolveConfigPath() string {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath != "" {
		return configPath
	}

	if _, err := os.Stat(".env"); err == nil {
		return ".env"
	}

	return ""
}

// loadDotEnv loads variables from .env only if they are not already set
// in the real process environment.
func loadDotEnv(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
