package profile

import (
	"path/filepath"
	"strings"
	"testing"
)

func makeSkillProfile(name, body, path string) *Profile {
	return &Profile{
		Name:    name,
		Body:    body,
		Path:    filepath.Join(path, "SKILL.md"),
		Source:  SourceGlobal,
		IsSkill: true,
		Inherits: false,
	}
}

func makeFlatProfile(name, body string) *Profile {
	return &Profile{
		Name:    name,
		Body:    body,
		Path:    "/profiles/" + name + ".md",
		Source:  SourceGlobal,
		IsSkill: false,
		Inherits: false,
	}
}

func TestComposeSkillProfile_HasBaseDirPrefix(t *testing.T) {
	skillDir := "/home/user/.brains/profiles/epic-planner"
	p := makeSkillProfile("epic-planner", "Skill body here", skillDir)

	profiles := map[string]*Profile{"epic-planner": p}
	composer := NewComposerWithSource(profiles, nil)

	result, err := composer.Compose([]string{"epic-planner"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPrefix := "Base directory for this skill: " + skillDir
	if !strings.HasPrefix(result.Content, wantPrefix) {
		t.Errorf("expected content to start with %q\ngot: %q", wantPrefix, result.Content[:min(100, len(result.Content))])
	}
	if !strings.Contains(result.Content, "Skill body here") {
		t.Error("expected skill body to be in output")
	}
}

func TestComposeFlatProfile_NoBaseDirPrefix(t *testing.T) {
	p := makeFlatProfile("flat-profile", "Flat body here")

	profiles := map[string]*Profile{"flat-profile": p}
	composer := NewComposerWithSource(profiles, nil)

	result, err := composer.Compose([]string{"flat-profile"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result.Content, "Base directory for this skill:") {
		t.Error("flat profile should not have base directory prefix")
	}
	if result.Content != "Flat body here" {
		t.Errorf("got %q, want %q", result.Content, "Flat body here")
	}
}

func TestComposeMDProfileIncludesSkill_HasBaseDirLine(t *testing.T) {
	skillDir := "/home/user/.brains/profiles/my-skill"
	skill := makeSkillProfile("my-skill", "Skill content", skillDir)
	wrapper := &Profile{
		Name:     "wrapper",
		Body:     "Wrapper body",
		Path:     "/profiles/wrapper.md",
		IsSkill:  false,
		Includes: []string{"my-skill"},
		Inherits: false,
	}

	profiles := map[string]*Profile{
		"my-skill": skill,
		"wrapper":  wrapper,
	}
	composer := NewComposerWithSource(profiles, nil)

	result, err := composer.Compose([]string{"wrapper"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "Base directory for this skill: "+skillDir) {
		t.Errorf("expected base dir line for included skill\ncontent: %q", result.Content)
	}
	if !strings.Contains(result.Content, "Wrapper body") {
		t.Error("expected wrapper body in output")
	}
}

func TestComposeTwoSkillProfiles_EachHasOwnBaseDirLine(t *testing.T) {
	dir1 := "/profiles/skill-one"
	dir2 := "/profiles/skill-two"

	profiles := map[string]*Profile{
		"skill-one": makeSkillProfile("skill-one", "Body one", dir1),
		"skill-two": makeSkillProfile("skill-two", "Body two", dir2),
	}
	composer := NewComposerWithSource(profiles, nil)

	result, err := composer.Compose([]string{"skill-one", "skill-two"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "Base directory for this skill: "+dir1) {
		t.Error("expected base dir line for skill-one")
	}
	if !strings.Contains(result.Content, "Base directory for this skill: "+dir2) {
		t.Error("expected base dir line for skill-two")
	}
}

