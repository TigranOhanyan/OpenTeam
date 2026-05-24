# Idempotency and Checkpoints

## The Problem: State and Suspension

In an agentic framework, execution is rarely instantaneous. An agent might need to wait for a database query to execute, a massive CI pipeline to finish, or a human to approve a destructive action. 

If the framework holds the execution state in memory while waiting, it becomes fragile. A server restart, a network timeout, or a long-running human-in-the-loop request will cause the entire thought process to be lost. 

To solve this, OpenTeam uses a **Durable Execution** model based on two core concepts: **Checkpoints** and **State-Machine Idempotency**.

## The Checkpoint (The Root Node)

Instead of trying to serialize and deserialize the exact line of code where the engine paused, OpenTeam relies on Checkpoints.

A Checkpoint is the root event that triggers a cascade of agentic work. Most commonly, this is a new message from a user. 

When the OpenTeam engine is invoked, it does not need to be told *where* it left off. It is simply pointed at a Checkpoint and told to resolve it. The engine begins walking down the execution tree (evaluating tasks, calling LLMs, parsing thoughts, delegating to sub-agents) starting from that root node.

## State-Machine Idempotency

To make starting from the Checkpoint efficient, the engine is strictly idempotent. 

True mathematical idempotency is impossible with non-deterministic LLMs, but OpenTeam implements **State-Machine Idempotency**. The golden rule of the engine is: *Never do expensive or external work without checking the database first.*

As the engine walks down the execution tree from the Checkpoint, it performs a check at every single node (every Step):
1. **Is this step already completed in the SQLite artifact?**
2. **If Yes:** Skip the execution. Load the result from the database, and instantly move to the next node.
3. **If No:** This is the **Frontier**. Execute the work (e.g., call the LLM), write the result to the database, and continue.

This means when the engine is re-run, it "fast-forwards" through the already completed parts of the thought process at disk-read speed until it hits the exact place it needs to resume work.

## How This Powers "Yield to Host"

This architecture makes yielding to external tools and Human-in-the-Loop (HITL) incredibly elegant. The engine does not need complex `Suspend()` or `Resume()` APIs. 

The flow works like this:

1. **The Engine Hits a Boundary:** The engine fast-forwards to the Frontier and discovers the LLM wants to call a tool or ask a human a question.
2. **The Yield:** The engine writes a `Pending` tool call or human request to the SQLite database. Because it cannot execute this itself, it simply shuts down and yields control to the host application.
3. **The Host Action:** The host application reads the SQLite database, sees the pending request, and handles it (e.g., executes an API call or waits for a user to click a button in a UI). Once finished, the host writes the result into the SQLite database and marks the request as `Completed`.
4. **The Resume:** The host application simply invokes the OpenTeam engine again, pointing it at the exact same Checkpoint.
5. **The Fast-Forward:** The engine starts at the root, fast-forwards through the initial LLM thoughts, sees the tool call is now `Completed`, loads the host's injected result, and hits the new Frontier (usually calling the LLM again to evaluate the tool result).

## Why This is the Holy Grail for Agents

By making the engine a replayable, idempotent state machine:

* **Crash Resilience:** If the server loses power mid-thought, nothing is lost. The next time the engine runs, it fast-forwards right back to the Frontier.
* **True Statelessness:** The Go process does not hold complex structs or execution stacks in memory while waiting for tools or humans. Memory can be safely garbage collected, and the next run can happen on a completely different physical server.
* **Time Travel Debugging:** Because the engine just replays the database, a developer can delete the last few rows of the SQLite database and re-run the engine to watch it make a different decision from that exact point in time.