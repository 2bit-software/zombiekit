# Initiative: gui-prompts-management

**Type**: feature
**Status**: complete
**Created**: 2026-01-21T11:06:33-08:00
**Completed**: 2026-01-21
**ID**: 697123b9-feature-gui-prompts-management

## Description

Add a unified "Prompts" section to the GUI that displays all prompt types (workflows, profiles, steps) in a single view with filtering, searching, and sorting capabilities.

## Goals

- Create a unified view for managing all prompt types
- Support filtering by category (workflow/profile/step) and source (local/global/embedded)
- Support text search across name and description
- Support sorting by name, category, or source
- Display detailed view for individual prompts with metadata and content

## Completion

**Completed**: 2026-01-21
**Duration**: ~6 hours (same day)

### Outcomes

| Work Item | Status | Notes |
|-----------|--------|-------|
| T001: workflow.List() | Complete | Added List method with source precedence |
| T002: Directory structure | Complete | Created prompts plugin structure |
| T003: Type definitions | Complete | PromptCategory, PromptSource, Prompt structs |
| T004: Converters | Complete | Workflow/profile/step to Prompt converters |
| T005: Plugin core | Complete | Plugin with SidebarItems, MountRoutes |
| T006: Filter/sort helpers | Complete | filterPrompts, sortPrompts functions |
| T007: List handler | Complete | GET /prompts with query params |
| T008: View handler | Complete | GET /prompts/{category}/{name} |
| T009: List template | Complete | HTMX-enabled list with filters |
| T010: View template | Complete | Detail view with markdown toggle |
| T011: Template registration | Complete | Embedded via go:embed |
| T012: Plugin registration | Complete | Registered in gui.go and gui_service.go |
| T013: Integration tests | Complete | 21 tests for prompts, 4 for workflow.List() |
| T014: E2E verification | Complete | All user stories verified |

### Files Changed

**New files:**
- `internal/webplugins/prompts/types.go`
- `internal/webplugins/prompts/converters.go`
- `internal/webplugins/prompts/handlers.go`
- `internal/webplugins/prompts/plugin.go`
- `internal/webplugins/prompts/plugin_test.go`
- `internal/webplugins/prompts/templates/list.html`
- `internal/webplugins/prompts/templates/view.html`

**Modified files:**
- `internal/workflow/service.go` - Added List() method
- `internal/workflow/service_test.go` - Added List() tests
- `internal/cli/gui.go` - Registered prompts plugin
- `internal/startup/gui_service.go` - Registered prompts plugin

### Test Coverage

- 25 total new tests
- All tests passing
- Full test suite continues to pass

### Notes

MVP (P1) scope completed:
- US1: Browse All Prompts
- US2: Filter and Search Prompts
- US3: View Prompt Details

P2 features (Create, Edit, Delete) not implemented - buttons present but disabled with "Coming soon" tooltip.
