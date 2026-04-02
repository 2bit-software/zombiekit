package rules

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
)

// ParseRule parses a rules file's content, extracting YAML frontmatter
// and the remaining markdown body.
func ParseRule(content []byte, name, filePath string, source RuleSource) (*Rule, error) {
	var fm RuleFrontmatter
	rest, err := frontmatter.Parse(bytes.NewReader(content), &fm)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", filePath, err)
	}

	body := strings.TrimSpace(string(rest))
	fileName := filepath.Base(filePath)
	ruleName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	if name == "" {
		name = ruleName
	}

	return &Rule{
		Name:     name,
		FileName: fileName,
		Source:   source,
		FilePath: filePath,
		Paths:    fm.NormalizedPaths(),
		Body:     body,
	}, nil
}
