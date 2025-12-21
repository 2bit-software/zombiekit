zombiekit
* has profiles which are just "sub-agents" or "skills" or list of rules
* can run a prompt like "/brains.run research,papi,database investigate how X works in blah"
	* it will then return a prompt that is the sum of research, papi, and database skills/agents
* zombiekit also a replacement for spec-kit
* new specs(/brains.feature, /brains.bug, /brains.refactor), /brains.plan, /brains.research /brains.tasks, /brains.analyze, /brains.clarify, /brains.implement, /brains.next
* can load domain-specific agents from agents.md file, claude.md, etc
* for each command, auto loads the correct supporting files to hyrdate the context
* for each command, it can also be run like "/brains.spec papi,database <thing>" that way we can make sure we use those domain specific agents to help with the spec
	* whatever we choose on the spec *should also* be used by other steps later on as well?
* commands order should be defined in another "sub-agent", for example:
	* for each command like "/brains.spec" we have a special profile that gets created using the "spec" context, as well as reading from the "<project_dir>/.brains/templates/<spec|research|etc-profile>.md"
* we source output artifacts as templates from "<project_dir>/.brains/templates/<spec|research|etc-artifact>.md"
* we walk up the tree from the calling direction to find .brains directories, up to either the git root or home directory, whichever comes first, and compose the agents with closest to CWD wins.
* when creatig a new spec, create a hex encoded unix timestamp of and name the spec "<hex>-title" where title is an AI summarised title of the work.
* the "specs" folder should exist in the root where claude code is being run from? or?

* profiles to make:
	* ai consumer auditor
	* separation of concerns auditor
	* researcher skill (top level skill that calls out to domain-experts, collates findings, returns)
	* spec creator skill (takes research, creates spec)
	* summarization of unique parts worth pointing out to the user
		* this could include an "allow list" of things that are common that we don't want to be notified about
		* interesting bits analyzer (analyze spec, plan, etc for interesting things to present to the user, this *can* be questions, but it can also illustrate unique decisions that were made)