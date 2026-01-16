package profile

import (
	"testing"
)

func TestParseFrontmatter_TypeField(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedType string
		expectedBody string
	}{
		{
			name: "action type",
			content: `---
name: test-profile
type: action
---
Content here`,
			expectedType: "action",
			expectedBody: "Content here",
		},
		{
			name: "domain type",
			content: `---
name: test-profile
type: domain
---
Content here`,
			expectedType: "domain",
			expectedBody: "Content here",
		},
		{
			name: "step type",
			content: `---
name: test-profile
type: step
---
Content here`,
			expectedType: "step",
			expectedBody: "Content here",
		},
		{
			name: "no type field",
			content: `---
name: test-profile
---
Content here`,
			expectedType: "",
			expectedBody: "Content here",
		},
		{
			name: "unknown type value",
			content: `---
name: test-profile
type: custom-type
---
Content here`,
			expectedType: "custom-type",
			expectedBody: "Content here",
		},
		{
			name: "uppercase type preserved",
			content: `---
name: test-profile
type: ACTION
---
Content here`,
			expectedType: "ACTION",
			expectedBody: "Content here",
		},
		{
			name: "mixed case type preserved",
			content: `---
name: test-profile
type: Domain
---
Content here`,
			expectedType: "Domain",
			expectedBody: "Content here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := ParseFrontmatter([]byte(tt.content))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fm.Type != tt.expectedType {
				t.Errorf("Type = %q, want %q", fm.Type, tt.expectedType)
			}

			if body != tt.expectedBody {
				t.Errorf("body = %q, want %q", body, tt.expectedBody)
			}
		})
	}
}

func TestParseProfile_TypeField(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedType string
	}{
		{
			name: "type field populated in profile",
			content: `---
name: test-profile
type: action
description: Test description
---
Content here`,
			expectedType: "action",
		},
		{
			name: "profile without type field",
			content: `---
name: test-profile
description: Test description
---
Content here`,
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := ParseProfile([]byte(tt.content), "fallback-name", "/test/path", SourceLocal)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if profile.Type != tt.expectedType {
				t.Errorf("Profile.Type = %q, want %q", profile.Type, tt.expectedType)
			}
		})
	}
}
