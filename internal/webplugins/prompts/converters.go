package prompts

import (
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

// Workflow converters

func convertWorkflow(wf *workflow.Workflow) Prompt {
	return Prompt{
		Name:        wf.Name,
		Category:    CategoryWorkflow,
		Source:      mapWorkflowSource(wf.Source),
		Description: wf.Description,
		Path:        wf.Path,
	}
}

func convertWorkflowFull(wf *workflow.Workflow) Prompt {
	p := convertWorkflow(wf)
	p.Content = wf.Content
	return p
}

func mapWorkflowSource(src string) PromptSource {
	switch src {
	case "local":
		return SourceLocal
	case "global":
		return SourceGlobal
	default:
		return SourceEmbedded
	}
}

// Profile converters

func convertProfile(entry profile.ListEntry) Prompt {
	return Prompt{
		Name:        entry.Name,
		Category:    CategoryProfile,
		Source:      mapProfileSource(entry.Source),
		Description: entry.Description,
		Path:        entry.Path,
		Shadowed:    entry.Shadowed,
		ProfileType: entry.Type,
		Includes:    entry.Includes,
		Inherits:    entry.Inherits,
		Model:       entry.Model,
		Color:       entry.Color,
	}
}

func convertProfileFull(result *profile.ShowResult) Prompt {
	return Prompt{
		Name:        result.Name,
		Category:    CategoryProfile,
		Source:      mapProfileSource(result.Source),
		Description: result.Description,
		Path:        result.Path,
		ProfileType: result.Type,
		Includes:    result.Includes,
		Inherits:    result.Inherits,
		Model:       result.Model,
		Color:       result.Color,
		Content:     result.Content,
	}
}

func mapProfileSource(src profile.ProfileSource) PromptSource {
	switch src {
	case profile.SourceLocal:
		return SourceLocal
	case profile.SourceGlobal:
		return SourceGlobal
	default:
		return SourceEmbedded
	}
}

// Step converters

func convertStep(s *step.Step) Prompt {
	return Prompt{
		Name:        s.Name,
		Category:    CategoryStep,
		Source:      mapStepSource(s.Source),
		Description: s.Description,
		Path:        s.Path,
		Profiles:    s.Profiles,
		Files:       s.Files,
	}
}

func convertStepFull(s *step.Step) Prompt {
	p := convertStep(s)
	p.Content = s.Directive
	return p
}

func mapStepSource(src step.StepSource) PromptSource {
	switch src {
	case step.SourceLocal:
		return SourceLocal
	case step.SourceGlobal:
		return SourceGlobal
	default:
		return SourceEmbedded
	}
}
