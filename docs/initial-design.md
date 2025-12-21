help me design the workflow that my new CLI application will have:

/brains.feature, /brains.bug, /brains.refactor
* performs the "spec" skill:
	1. sorts out "business spec" from "technical spec" from user, saves "technical spec" for next command
	2. research (many agents)
	3. spec creation (single agent)
	4. audit (many agents)
	5. highlight to user

/brains.plan
* performs the the "plan" skill
	1. loads the previous "technical spec" if any
	2. research (many agents)
	3. plan creation (single agent)
	4. audit (many agents)
	5. highlight to user

/brains.tasks
* performs the "tasks" skill
	1. creates the necessary task list
	2. breaks the user stories down into sub-components that don't rely on each other and can start being implemented from a fresh context

/brains.implement
	1. for each user story that has it's own context, implement using the specified agents

/brains.research
	* perform iterative, repeatable research on a subject

/brains.update (or .alter, or .change or something)
	* updates the original spec, whatever it was

/brains.clarify
	* audit the spec and/or plan for clarifications and things that need updating

/brains.audit
	* makes sure all artifacts are aligned

/brains.eat
	* fun command which i'm not sure what it does yet
