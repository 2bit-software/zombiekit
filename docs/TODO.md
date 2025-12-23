* would be good to have a way to calculate if a set of tasks is too complicated, and break it up into multiple task lists
* single command that starts up:
	* mcp server
	* web server
	* conversation importer
* need to design the interface to communicate to LLM for vector generation
* also want to make sure we document that during initial spec creation, we have a process that:
	* takes OUT technical implementations provided by user and research agents, and dumps it into technical_requirements_research.md instead. Later, during the planning/technical implementation phase, we utilize this document
* the doc states: - ❌ Make LLM API calls (Claude Code does this) however, we *will* call an LLM to generate vector embeddings. Right now, most likely a local ollama instance.
* assume server is SSE by default
* agent which proves out the plan in TDD/small unit/test cases, so that by the time we are implementing, there are NO gotchas. This is *basically* implementing, so what do we want here?
	* maybe a first attempt at implementing incrementally, learn lessons, and plan/try again?
* I also want a "fast" feature, where maybe the planning and specs are less thorough?
* one thing i'd like is to de-couple implementing multiple "related" but not technically reliant sets of changes. So for example, I was making a new CLI, and wanted to have the version displayed while building. This required also adding the version library and build flags for versions to be embedded. These are two separate things. So i'd like for there to be two different specs, which *may* have different testing/researching requirements
	* the above makes me feel like we should group "change sets" by request, and allow there to be multiple "spec" or even "refactor" operations within that same "umbrella" initiative.
	* so maybe the root folder is "history", then "timestamp-<summary of project>"
	* and inside, until the project is considered complete, you could have multiple specs, refactors, and bugs. The first initial spec is often not complete, or i'm unhappy with the result, so want to refactor, or I catch bugs and want to update
	* this makes me feel like then we should have a document at the root of each "project" or "initiative" that describes the original (and updated) goal of the work, and anything that could be useful for any of the individual specs/bugs/refactor work within that project