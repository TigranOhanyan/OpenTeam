# Implementation Architecture

OpenTeam is designed to be a highly predictable, stateless execution engine. It uses a centralized, deterministic Agentic Loop operating over a portable state artifact.

## 1. The State Layer: Ephemeral SQLite (The Portable Artifact)

OpenTeam does not require a central, stateful database (like Postgres or Redis) to run. Instead, it utilizes a **"Shared Nothing" / Request-Isolated** architecture.

*   **Per-Request Isolation:** Each incoming request provides its own state. The framework spins up a temporary SQLite database (e.g., in-memory or `/tmp/{request_id}.db`) for the duration of the request.
*   **The Artifact:** This SQLite file contains the entire OpenTeam schema: the Team, the Members, the Channels, the Roles, and the Message history.
*   **Stateless Principle:** The OpenTeam engine itself is fully stateless. It takes an SQLite file as input, runs the execution loop, mutates the SQLite file, and returns it. The host application is responsible for persisting this file (e.g., to S3 or a local disk) between turns.

This approach provides perfect "Time Travel" debugging. If a conversation fails, a developer can download the exact `.sqlite` file, feed it into the local OpenTeam engine, and reproduce the exact state.

## 2. The Logic Layer: The Centralized Engine (OpenTeam)

To ensure predictable execution and easy debugging, OpenTeam uses a **Centralized Agentic Engine**. 

This engine acts as the "Operating System" for the team. It is an orchestrator that reads the state, decides who needs to act, and coordinates the flow of information, supporting both sequential duties and parallel execution.

### How the Engine Works
There is one active orchestrator per request. It acts as the scheduler for the organization.

1.  **Read State:** The engine queries the SQLite database to find all pending actions (e.g., new messages in Channels, pending Duties, or unresolved tool calls).
2.  **Determine Active Members:** Based on the state, the engine identifies which Members need to act. Multiple Members can be active simultaneously.
3.  **Execute Duties (Parallel & Sequential):** 
    *   *Intra-Agent:* An individual Member's Duties are strictly sequential (e.g., Security must pass before Worker starts).
    *   *Inter-Agent:* Multiple active Members run their respective Duties in parallel.
4.  **Process Protocol Tools:** 
    *   If an LLM calls `ask_member` (even for multiple peers), the engine spawns concurrent tasks for the target Members.
    *   If an LLM calls `liaise_to_channel`, the engine writes the summary to the target Channel and wakes up the relevant Member in that Channel concurrently.
    *   If an LLM calls `pass_to_next_duty`, the engine advances that specific Member's internal state.
5.  **Process Expert Tools:** If an LLM calls domain tools (e.g., `query_db`, `search_web`), the engine executes these Go functions concurrently, synchronizes the results back to the SQLite state, and resumes the Member.
6.  **Yield:** Once all Members have finished their Duties and no further tasks are pending, the engine terminates and returns the final state.

## 3. Why This Wins

1.  **Determinism:** Because the central engine manages the state mutations, the execution path is highly predictable. If you provide the same SQLite file and the LLM returns the same tool calls, the outcome is consistent.
2.  **Simplicity:** The framework abstracts away the complexity of routing. It simply reads the database, invokes the necessary Members, and updates the database. 
3.  **Auditability:** Every step of the engine (every Duty shift, every tool call) is written to the SQLite file. The resulting file is a perfect, linear audit log of the entire organizational thought process.