package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsRule_ArrayPaths(t *testing.T) {
	content := []byte(`---
paths:
  - "**/*.go"
  - "**/*.mod"
---
# Go Standards

- Use any
`)
	rule, err := ParseRule(content, "", "/tmp/go.md", SourceProject)
	require.NoError(t, err)
	assert.Equal(t, "go", rule.Name)
	assert.Equal(t, "go.md", rule.FileName)
	assert.Equal(t, SourceProject, rule.Source)
	assert.Equal(t, []string{"**/*.go", "**/*.mod"}, rule.Paths)
	assert.Equal(t, "# Go Standards\n\n- Use any", rule.Body)
	assert.False(t, rule.IsUnconditional())
}

func TestParseRule_StringPath(t *testing.T) {
	content := []byte(`---
paths: "**/*.go"
---
# Go
`)
	rule, err := ParseRule(content, "", "/tmp/go.md", SourceGlobal)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*.go"}, rule.Paths)
}

func TestParseRule_NoFrontmatter(t *testing.T) {
	content := []byte(`# General Rules

- Always check errors
`)
	rule, err := ParseRule(content, "", "/tmp/general.md", SourceProject)
	require.NoError(t, err)
	assert.Nil(t, rule.Paths)
	assert.True(t, rule.IsUnconditional())
	assert.Equal(t, "# General Rules\n\n- Always check errors", rule.Body)
}

func TestParseRule_EmptyBody(t *testing.T) {
	content := []byte(`---
paths:
  - "**/*.go"
---
`)
	rule, err := ParseRule(content, "", "/tmp/go.md", SourceProject)
	require.NoError(t, err)
	assert.Empty(t, rule.Body)
}

func TestParseRule_EmptyPaths(t *testing.T) {
	content := []byte(`---
paths: []
---
# Content
`)
	rule, err := ParseRule(content, "", "/tmp/test.md", SourceProject)
	require.NoError(t, err)
	assert.Nil(t, rule.Paths)
	assert.True(t, rule.IsUnconditional())
}

func TestNormalizedPaths_Nil(t *testing.T) {
	fm := RuleFrontmatter{Paths: nil}
	assert.Nil(t, fm.NormalizedPaths())
}

func TestNormalizedPaths_EmptyString(t *testing.T) {
	fm := RuleFrontmatter{Paths: ""}
	assert.Nil(t, fm.NormalizedPaths())
}

func TestRuleID(t *testing.T) {
	rule := &Rule{FileName: "go.md", Source: SourceProject}
	assert.Equal(t, "project:go.md", rule.ID())

	rule2 := &Rule{FileName: "go.md", Source: SourceGlobal}
	assert.Equal(t, "global:go.md", rule2.ID())
}
