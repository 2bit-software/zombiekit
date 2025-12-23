zombiekit:
* mcp server for storing memory using "sticky memory" system via streamable
* mcp server for "code thinking" via streamable
* automatic import of claude conversations for full text and vectorized search (side process, not mcp or hook)
* claude code commands analogous to "spec-kit" with agents for e2e development workflow
* system to support "prompts" management
* cli endpoint that allows composition of agents, like "./zombiekit prompts combine research,database" will return the combined prompts for "research" and "database" agents
* web frontend that allows managing all these functionalities

### Profile Design

profiles are stored using the profile name. They are hierarchical, so repo-specific profiles are stored locally, and then globally at ~/.brains/...

each "override" profile can specify whether it inherits the base version of that profile (like the consitution, for example), or it overrides completely.

This means that when you call "./zombiekit prompts combine <profiles>" it needs to know which project/directory it is being called from, so it can try to find prompts of that name within the correct folder structure.

Also, prompts need to be "importable" from claude code's agents/skills directories.

This means, that to create a web GUI that allows us to manage these profiles, zombiekit needs to create a list of repos/file paths that contain .brains folders that contain profiles. We should add new directories to this list either intentionally via the webgui, intentionally via some command like /zombiekit prompts add <location> or if we call "./zombiekit prompts combine <profiles>" from a new location

	