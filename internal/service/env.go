package service

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EnvFileInfo describes an env file found in the repo.
type EnvFileInfo struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// EnvVariable describes an env variable and its state.
type EnvVariable struct {
	Key   string `json:"key"`
	State string `json:"state"` // set|empty|placeholder
	Note  string `json:"note,omitempty"`
}

// EnvResult holds the environment variable inspection result.
type EnvResult struct {
	EnvFiles  []EnvFileInfo `json:"env_files"`
	Variables []EnvVariable `json:"variables"`
	Warnings  []string      `json:"warnings"`
}

// EnvService inspects environment variable configuration.
type EnvService struct {
	repoRoot string
}

// NewEnvService creates an EnvService.
func NewEnvService(repoRoot string) *EnvService {
	return &EnvService{repoRoot: repoRoot}
}

// placeholder patterns — values that look like placeholders.
var placeholderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^x+$`),
	regexp.MustCompile(`(?i)^YOUR_`),
	regexp.MustCompile(`^<[^>]+>$`),
	regexp.MustCompile(`(?i)^TODO$`),
	regexp.MustCompile(`(?i)^changeme$`),
	regexp.MustCompile(`(?i)^dummy`),
	regexp.MustCompile(`(?i)^fake`),
	regexp.MustCompile(`(?i)^example`),
	regexp.MustCompile(`^0+$`),
	regexp.MustCompile(`^\*+$`),
	regexp.MustCompile(`(?i)^REPLACE_`),
}

// Inspect reads .env files and classifies variable states.
func (e *EnvService) Inspect(_ context.Context) (*EnvResult, error) {
	result := &EnvResult{}

	candidates := []string{
		".env",
		".env.local",
		".env.example",
		".env.template",
		".env.development",
		".env.production",
	}

	allVars := make(map[string]string)

	for _, name := range candidates {
		absPath := filepath.Join(e.repoRoot, name)
		_, err := os.Stat(absPath)
		exists := err == nil
		result.EnvFiles = append(result.EnvFiles, EnvFileInfo{Path: name, Exists: exists})

		if !exists {
			continue
		}
		vars, err := parseEnvFile(absPath)
		if err != nil {
			continue
		}
		for k, v := range vars {
			if _, seen := allVars[k]; !seen {
				allVars[k] = v
			}
		}
	}

	for key, value := range allVars {
		ev := classifyVar(key, value)
		result.Variables = append(result.Variables, ev)
		switch ev.State {
		case "placeholder":
			result.Warnings = append(result.Warnings, key+" is a placeholder — likely causing authentication failures")
		case "empty":
			result.Warnings = append(result.Warnings, key+" is empty — dependent features will fail")
		}
	}

	return result, nil
}

func classifyVar(key, value string) EnvVariable {
	ev := EnvVariable{Key: key}
	if value == "" {
		ev.State = "empty"
		return ev
	}
	for _, re := range placeholderPatterns {
		if re.MatchString(value) {
			ev.State = "placeholder"
			ev.Note = "value matches placeholder pattern"
			return ev
		}
	}
	ev.State = "set"
	return ev
}

func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if len(value) >= 2 &&
			((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		vars[key] = value
	}
	return vars, scanner.Err()
}
