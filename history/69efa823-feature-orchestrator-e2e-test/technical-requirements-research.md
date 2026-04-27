# Technical Requirements Research: Orchestrator E2E Test

## Implementation Hints from Ticket

- The test doesn't need to be a Go test — a shell script or standalone Go binary is acceptable
- Sessions should be stubbed or immediately killed
- Run against dedicated test Linear project / throwaway GitHub repo (if using real APIs)
- Pre-seed `.ai/pr-description.md` in worktree, fire synthetic callbacks via HTTP POST

## Technical Preferences

- Prefer in-process mock clients over real API calls (CI-friendly, no credentials needed)
- Use existing `MockClient` implementations for Linear and GitHub
- Standard `go test` with build tag (`// +build integration` or `// +build e2e`)
- Manual poll-loop driving (call watcher tick functions directly) rather than relying on timers
- Single test file in `internal/orchestrator/` or `tests/e2e/`

## Research Needed

- [ ] Current mock client capabilities — do they record call history well enough to assert state transitions?
- [ ] How watchers expose their poll loop — can we call a single tick?
- [ ] How the callback server is initialized — can we drive it with test HTTP requests?
- [ ] State store inspection — can we query job status, watermarks, slots directly?
- [ ] Worktree manager — can we use a temp git repo for real worktree operations?
- [ ] Reconciliation — how is it triggered and what does it assert?
