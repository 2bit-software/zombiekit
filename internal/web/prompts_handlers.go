package web

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

type promptsHandlers struct {
	profileSvc  *profile.Service
	stepSvc     *step.Service
	workflowSvc *workflow.Service
}

func newPromptsHandlers(profileSvc *profile.Service, stepSvc *step.Service, workflowSvc *workflow.Service) *promptsHandlers {
	return &promptsHandlers{
		profileSvc:  profileSvc,
		stepSvc:     stepSvc,
		workflowSvc: workflowSvc,
	}
}

func (h *promptsHandlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	opts := PromptsFilterOptions{
		Category: r.URL.Query().Get("category"),
		Source:   r.URL.Query().Get("source"),
		Query:    r.URL.Query().Get("q"),
	}
	sortOpts := PromptsSortOptions{
		Field: r.URL.Query().Get("sort"),
		Order: r.URL.Query().Get("order"),
	}
	if sortOpts.Field == "" {
		sortOpts.Field = "name"
	}
	if sortOpts.Order == "" {
		sortOpts.Order = "asc"
	}

	// Aggregate prompts from all sources
	prompts, err := h.aggregatePrompts()
	if err != nil {
		data := PromptsListData{Error: err.Error()}
		renderer.Render(w, r, "prompts/list.html", data)
		return
	}

	// Filter
	prompts = filterPrompts(prompts, opts)

	// Sort
	sortPrompts(prompts, sortOpts)

	data := PromptsListData{
		Prompts:        prompts,
		CategoryFilter: opts.Category,
		SourceFilter:   opts.Source,
		Query:          opts.Query,
		SortField:      sortOpts.Field,
		SortOrder:      sortOpts.Order,
	}

	if err := renderer.Render(w, r, "prompts/list.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *promptsHandlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	category := chi.URLParam(r, "category")
	name := chi.URLParam(r, "name")

	prompt, err := h.getPrompt(PromptCategory(category), name)
	if err != nil {
		data := PromptsViewData{Error: err.Error()}
		renderer.Render(w, r, "prompts/view.html", data)
		return
	}

	data := PromptsViewData{Prompt: prompt}
	if err := renderer.Render(w, r, "prompts/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *promptsHandlers) aggregatePrompts() ([]Prompt, error) {
	var prompts []Prompt

	if h.workflowSvc != nil {
		if workflows, err := h.workflowSvc.List(); err == nil {
			for _, wf := range workflows {
				prompts = append(prompts, convertWorkflow(wf))
			}
		}
	}

	if h.profileSvc != nil {
		if profiles, err := h.profileSvc.List(); err == nil {
			for _, p := range profiles {
				prompts = append(prompts, convertProfile(p))
			}
		}
	}

	if h.stepSvc != nil {
		if steps, err := h.stepSvc.ListSteps(); err == nil {
			for _, s := range steps {
				prompts = append(prompts, convertStep(s))
			}
		}
	}

	return prompts, nil
}

func (h *promptsHandlers) getPrompt(category PromptCategory, name string) (*Prompt, error) {
	switch category {
	case CategoryWorkflow:
		if h.workflowSvc == nil {
			return nil, fmt.Errorf("workflow service not available")
		}
		wf, err := h.workflowSvc.Load(name)
		if err != nil {
			return nil, err
		}
		p := convertWorkflowFull(wf)
		return &p, nil

	case CategoryProfile:
		if h.profileSvc == nil {
			return nil, fmt.Errorf("profile service not available")
		}
		result, err := h.profileSvc.Show(name, false)
		if err != nil {
			return nil, err
		}
		p := convertProfileFull(result)
		return &p, nil

	case CategoryStep:
		if h.stepSvc == nil {
			return nil, fmt.Errorf("step service not available")
		}
		s, err := h.stepSvc.GetStep(name)
		if err != nil {
			return nil, err
		}
		p := convertStepFull(s)
		return &p, nil

	default:
		return nil, fmt.Errorf("unknown category: %s", category)
	}
}

// Filter and sort helpers

func filterPrompts(prompts []Prompt, opts PromptsFilterOptions) []Prompt {
	var filtered []Prompt

	for _, p := range prompts {
		// Category filter
		if opts.Category != "" && string(p.Category) != opts.Category {
			continue
		}

		// Source filter
		if opts.Source != "" && string(p.Source) != opts.Source {
			continue
		}

		// Query filter (search name and description)
		if opts.Query != "" {
			query := strings.ToLower(opts.Query)
			name := strings.ToLower(p.Name)
			desc := strings.ToLower(p.Description)
			if !strings.Contains(name, query) && !strings.Contains(desc, query) {
				continue
			}
		}

		filtered = append(filtered, p)
	}

	return filtered
}

func sortPrompts(prompts []Prompt, opts PromptsSortOptions) {
	sort.Slice(prompts, func(i, j int) bool {
		var cmp int
		switch opts.Field {
		case "name":
			cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
		case "category":
			cmp = strings.Compare(string(prompts[i].Category), string(prompts[j].Category))
			if cmp == 0 {
				cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
			}
		case "source":
			cmp = strings.Compare(string(prompts[i].Source), string(prompts[j].Source))
			if cmp == 0 {
				cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
			}
		default:
			cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
		}

		if opts.Order == "desc" {
			return cmp > 0
		}
		return cmp < 0
	})
}
