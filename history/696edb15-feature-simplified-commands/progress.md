# Progress Log

## T001 - Add ComposeOptions struct
- Status: Complete
- Files: internal/profile/service.go
- Notes: Added `ComposeOptions` struct with `WorkflowOnly` field

## T002 - Add ComposeWithOptions method
- Status: Complete
- Files: internal/profile/service.go
- Notes: Added `ComposeWithOptions` method and `filterByType` helper that filters profiles by type

## T003 - Add workflow parameter to HandleCompose
- Status: Complete
- Files: internal/mcp/tools/profile/tool.go, internal/mcp/server.go
- Notes: Added `workflow` boolean parameter to HandleCompose and registered it in MCP schema

## T004 - Unit test for workflow filter in service
- Status: Complete
- Files: internal/profile/service_test.go
- Notes: Added `TestProfileService_ComposeWithOptions` with 4 test cases

## T005 - Unit test for MCP tool workflow parameter
- Status: Complete
- Files: internal/mcp/tools/profile/tool_test.go
- Notes: Added `TestMCPTool_HandleCompose_WorkflowFilter` with 3 test cases

## T006 - Create workflow profile
- Status: Complete
- Files: profiles/new.md
- Notes: Created with `type: workflow` and classification instructions

## T007 - Create brains.new command
- Status: Complete
- Files: integrations/claude/commands/brains.new.md
- Notes: Calls profile-compose with `workflow: true`

## T008-T010 - Delete legacy embedded commands
- Status: Complete
- Files: Deleted integrations/claude/commands/brains.{feature,bug,refactor}.md
- Notes: Removed 3 legacy commands

## T011 - Update init tests
- Status: Complete
- Files: internal/cli/init_test.go
- Notes: Updated references to brains.new.md and file count from 16 to 14

## T012-T015 - Local command cleanup
- Status: Complete
- Files: .claude/commands/brains.{feature,bug,refactor}.md deleted, brains.new.md created
- Notes: Synced local commands with embedded commands
