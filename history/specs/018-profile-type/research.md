# Research: Profile Type Classification

**Feature**: 018-profile-type
**Date**: 2025-12-23

## Research Questions

### 1. Existing Pattern for Optional Fields in Profile Frontmatter

**Question**: How are optional fields like `model` and `color` currently implemented?

**Finding**: Examined `internal/profile/types.go`:
- `ProfileFrontmatter` uses simple string fields: `Model string` and `Color string`
- `Profile` struct mirrors these with `Model string` and `Color string`
- `ListEntry` and `ShowResult` include these with `json:"model,omitempty"` tags
- The `omitempty` JSON tag handles the optional nature

**Decision**: Follow identical pattern for `Type` field
**Rationale**: Consistency with existing codebase, no learning curve
**Alternatives Considered**: Enum type with validation - rejected as over-engineering for 3 values

### 2. Web UI Badge Display Pattern

**Question**: How are badges/tags displayed in the profiles list?

**Finding**: Examined `internal/webplugins/profiles/templates/list.html`:
- Uses `<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium">` pattern
- Different color schemes: `bg-yellow-100 text-yellow-800` for shadowed, `bg-gray-100 text-gray-600` for source
- Badges appear inside the profile name `<div>` with `space-x-2` for spacing

**Decision**: Add type badge with color-coded scheme:
- Action: `bg-purple-100 text-purple-800`
- Domain: `bg-green-100 text-green-800`
- Step: `bg-blue-100 text-blue-800`
- Unknown types: `bg-gray-100 text-gray-600`

**Rationale**: Distinct colors help visual differentiation; follows Tailwind CSS conventions already in use
**Alternatives Considered**: Single color with text label only - rejected as less scannable

### 3. Case Sensitivity Handling

**Question**: How should case sensitivity work for type values?

**Finding**: The spec requires:
- Case-insensitive matching for known types
- Preserve original casing for display

**Decision**: Store the original value as-is in the `Type` field. UI templates will use case-insensitive comparison for color selection but display the original value.

**Rationale**: Simplest implementation - no transformation at parse time, template handles display logic
**Alternatives Considered**: Normalize to lowercase at parse time - rejected as it loses user's intended display format

### 4. Unknown Type Values

**Question**: How to handle type values that aren't action/domain/step?

**Finding**: Spec requires accepting unknown values for forward compatibility (FR-008)

**Decision**: Parse and store any string value. Known types get colored badges; unknown types get gray/neutral styling.

**Rationale**: Forward compatibility allows adding new types without breaking existing profiles
**Alternatives Considered**: Log warning for unknown types - deferred, may add later if needed

## No Outstanding Unknowns

All technical decisions resolved. Ready for Phase 1 design.
