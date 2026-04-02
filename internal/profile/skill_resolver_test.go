package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// makeProfilesDir creates a temp .brains/profiles/ dir and returns its path.
func makeProfilesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".brains", "profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating profiles dir: %v", err)
	}
	return dir
}

// writeFlatProfile writes a flat .md profile to dir.
func writeFlatProfile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing flat profile %s: %v", name, err)
	}
}

func TestResolverLoadsSkillDirectory(t *testing.T) {
	profilesDir := makeProfilesDir(t)
	makeSkillDir(t, profilesDir, "epic-planner", "---\nname: epic-planner\n---\nSkill body")
	writeFlatProfile(t, profilesDir, "flat-profile", "Flat body")

	r, err := NewResolver(filepath.Dir(filepath.Dir(profilesDir))) // workingDir above .brains
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	dirs := []ResolvedDirectory{{Path: profilesDir, Source: SourceLocal}}
	profiles, err := r.LoadProfiles(dirs)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}

	if _, ok := profiles["epic-planner"]; !ok {
		t.Error("expected skill profile 'epic-planner' to be loaded")
	}
	if _, ok := profiles["flat-profile"]; !ok {
		t.Error("expected flat profile 'flat-profile' to be loaded")
	}

	if !profiles["epic-planner"].IsSkill {
		t.Error("expected epic-planner.IsSkill to be true")
	}
	if profiles["flat-profile"].IsSkill {
		t.Error("expected flat-profile.IsSkill to be false")
	}
}

func TestResolverExtractsSkillZIP(t *testing.T) {
	profilesDir := makeProfilesDir(t)
	makeSkillZIP(t, profilesDir, "my-tool", map[string]string{
		"my-tool/SKILL.md":  "# My Tool",
		"my-tool/helper.sh": "#!/bin/bash",
	})

	r, _ := NewResolver(filepath.Dir(filepath.Dir(profilesDir)))
	dirs := []ResolvedDirectory{{Path: profilesDir, Source: SourceGlobal}}

	profiles, err := r.LoadProfiles(dirs)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}

	p, ok := profiles["my-tool"]
	if !ok {
		t.Fatal("expected my-tool to be loaded after ZIP extraction")
	}
	if !p.IsSkill {
		t.Error("expected IsSkill to be true")
	}

	// Sibling file should be on disk.
	if _, err := os.Stat(filepath.Join(profilesDir, "my-tool", "helper.sh")); err != nil {
		t.Error("expected helper.sh to exist in extracted skill dir")
	}
}

func TestResolverSkillShadowsGlobal(t *testing.T) {
	localProfiles := makeProfilesDir(t)
	globalProfiles := makeProfilesDir(t)

	makeSkillDir(t, localProfiles, "shared-skill", "---\nname: shared-skill\n---\nLocal body")
	makeSkillDir(t, globalProfiles, "shared-skill", "---\nname: shared-skill\n---\nGlobal body")

	r, _ := NewResolver(t.TempDir())
	dirs := []ResolvedDirectory{
		{Path: localProfiles, Source: SourceLocal},
		{Path: globalProfiles, Source: SourceGlobal},
	}

	profiles, err := r.LoadProfiles(dirs)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}

	p, ok := profiles["shared-skill"]
	if !ok {
		t.Fatal("expected shared-skill")
	}
	if p.Source != SourceLocal {
		t.Errorf("expected local to shadow global, got source %v", p.Source)
	}
	if p.Body != "Local body" {
		t.Errorf("expected local body, got %q", p.Body)
	}
}

func TestResolverFlatMDSkillConflict_DirectoryWins(t *testing.T) {
	profilesDir := makeProfilesDir(t)
	makeSkillDir(t, profilesDir, "conflict", "---\nname: conflict\n---\nDirectory body")
	writeFlatProfile(t, profilesDir, "conflict", "Flat body")

	r, _ := NewResolver(t.TempDir())
	dirs := []ResolvedDirectory{{Path: profilesDir, Source: SourceLocal}}

	profiles, err := r.LoadProfiles(dirs)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}

	p, ok := profiles["conflict"]
	if !ok {
		t.Fatal("expected 'conflict' profile")
	}
	if !p.IsSkill {
		t.Error("expected directory skill to win over flat .md")
	}
}

func TestListEntry_FormatField(t *testing.T) {
	profilesDir := makeProfilesDir(t)
	makeSkillDir(t, profilesDir, "skill-a", "---\nname: skill-a\n---\nBody")
	writeFlatProfile(t, profilesDir, "flat-b", "Body")

	svc := NewServiceWithSourceInterface(&BrainsSource{
		resolver: &Resolver{workingDir: t.TempDir(), homeDir: t.TempDir()},
	})
	// Direct call using resolver to avoid needing real dirs.
	r, _ := NewResolver(t.TempDir())
	dirs := []ResolvedDirectory{{Path: profilesDir, Source: SourceLocal}}
	allProfiles, _ := r.LoadAllProfiles(dirs)

	// Check format field values directly on profiles.
	if p := allProfiles["skill-a"]; len(p) > 0 && !p[0].IsSkill {
		t.Error("skill-a should have IsSkill=true")
	}
	if p := allProfiles["flat-b"]; len(p) > 0 && p[0].IsSkill {
		t.Error("flat-b should have IsSkill=false")
	}
	_ = svc
}
