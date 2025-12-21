I want to create a new tool, written in Go. It should be it's own tool/<cmd> entrypoint. It will:
* constantly import conversations as they happen from claude code
* chunk them appropriately
* save to a postgres database
* categorize who it came from
* categorize who kind of thing it is:
	* original system prompt?
	* ai response?
	* client question/followup?
	* a link to a file/url?
* if a link to a file via local disk or url, we should load that file/document and save it (if not saved already)
	* which means we need to normalize and version these files. Maybe sha local files, and check for diffs on pages or something?
* save the whole text of the conversation to a full text search column
* also get the vectorized embedding from a local LLM and save that
* the tool should have a Taskfile to start up the requirements
* postgres in docker
* tool also built and served via docker or locally
	* which means docker container needs claude path mapped into container so it can read/import





