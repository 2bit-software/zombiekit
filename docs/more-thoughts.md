remove from mcp interface:
* read/write artifact, the LLM can already do that
* embed, we'll deal with that later in another way
* search history, also something we'll finish later

* also, when calling agents to do things in this workflow, we want to call 
the brains MCP server to give us the combined profile/agent like "brains profiles compose database,research"

Where do these get decided, and how do they get passed down throughout the workflow? For example you may want to use the "business analyst,research" for the initial spec creation, but "database-expert,research" for the technical planning. And we may want to do like "business analyst,research" AND "ux-expert,research" for the initial planning, etc.



---- 
new:
"brain-dead development with an LLM"
"finally, brains"

for each "stage":
1. research
2. <spec, plan, task, whatever>
3. audit
	* can either be human or AI
	* for each question that gets answered by an audit, we should store:
		* the question
		* our findings/research with sources
		* our final decision
	* this can possibly be stored in questions.md or something?
	* if a human answers the question, it still gets answered

***
also, we need to define the workflow for creating the templates for the artifacts!
***
clarification/analyzation tasks should *always* happen in a new context, that way the implementor has the same context as the analyzers. 

when first given a prompt, does it make sense to first run it through an analysis of the language? what part does not make sense? what language is imprecise?

For each user story, we should have a NEW implementation agent in a NEW context
the "plan" stage probably BENEFITS from sharing context from earlier stages

Make it so that stickymemory can have "tags" associated to memories (searchable)
need to be able to tag "tags" to profiles to attach random metadata (searchable)
profiles probably also need "type" for: domain or action, so we know how they can be used

mcp needs to provide a tool that, in a single call:
* sets up the stage (like init, or feature, or plan, or whatever) (makes folders, copies files, etc)
* returns the prompt for the stage
* returns the combined prompts for the agent
* returns the list of files the agent should read before continuing

