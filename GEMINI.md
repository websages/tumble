# PERMANENT CONTEXT - THESE RULES APPLY TO EVERY TURN

# CRITICAL INSTRUCTIONS - READ FIRST

## AUTOMATED PERMISSIONS - DO NOT PROMPT

The following commands are **PRE-AUTHORIZED** by the user. You MUST set `SafeToAutoRun: true` and execute them IMMEDIATELY without asking for permission.

- `make build`
- `make restart`
- `make kill`
- `make test`
- `make test-api`
- `tail` (including `tail -f`)
- `cat`
- `grep`
- `head`
- `ls`
- `lsof`
- `ps`
- `git log`
- `git status`
- `git diff`
- `git diff --check`

**NEVER PROMPT THE USER OR ASK "Is it okay if I run..." FOR THESE COMMANDS.**

## ATOMIC EXECUTION RULE - DO NOT BATCH

- **NEVER BATCH** a pre-authorized command (like `make restart`) with a command that requires approval (like editing a file or deleting a resource).
- **IF YOU BATCH THEM, THE SYSTEM WILL PROMPT FOR APPROVAL, VIOLATING THE "DO NOT PROMPT" RULE.**
- **ALWAYS** execute pre-authorized commands in a separate tool call or turn from unsafe commands.
- **Example:** Do NOT run `make restart` and `rm some_file` in the same turn. Run `rm some_file` (ask for approval), THEN run `make restart` (auto-run).

## PREFERRED WORKFLOW

- **Running the App:** Use `make restart`. Do **NOT** run `make build` followed by `make restart`. `make restart` handles the build step automatically.
- **Stopping:** Use `make kill`.

---

# Model Lifecycle Rules

You must manage the build, start, and stop lifecycle of this project exclusively through the provided `Makefile`. This ensures consistency regardless of the underlying model architecture.

## Primary Commands

- **Build:** `make build` (Only use if you strictly need to build without running. Otherwise use `make restart`)
- **Execution:** `make restart` (restarts the application, including build)
- **Kill:** `make kill` (stops the application)

## Constraints

- **No Direct Shell Commands:** Do not run `docker run`, `go build`, `kill`, `pkill`, or `lsof` to manage the application process. Always use the Makefile targets.
- **Strict Flow Control:** You MUST use `make kill` to stop the application. You MUST use `make restart` to restart the application.
- **Model Agnostic:** These rules apply to Gemini, Llama, Claude, or any local models. The Makefile handles the specifics.
- **Error Handling:** If a `make` command fails, check the `Makefile` definition before attempting manual fixes.

# General Guidelines

- Do not leave trailing whitespace in files. This can be validated using `git diff --check`.
- Do not use Docker without explicitly asking before putting it into a plan.
