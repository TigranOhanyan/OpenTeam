# Tools and Human-in-the-Loop

## The Execution Model

When designing how an agentic framework handles external actions (Tool Calls) and human intervention (Human-in-the-Loop or HITL), there are generally two architectural paths:

**The "Batteries Included" Network Engine**
The framework itself handles the execution. It manages network requests, tool discovery, authentication, timeouts, and retries. When an LLM calls a tool, the framework executes it internally and continues the loop.

**The "Yield to Host" State Machine**
The framework acts purely as a state machine. When an LLM decides to call a tool or ask a human a question, the framework halts execution, writes the pending request to the state artifact, and yields control back to the host application. The host application is responsible for fulfilling the request and resuming the engine.

## The Decision: Yield to Host

OpenTeam adopts a **Yield to Host** architecture. The core engine does not execute tools, nor does it manage websockets for human input. It strictly yields execution to the host application.

This decision is driven by the core architectural principles of the framework: the **Stateless Engine** and the **Portable SQLite Artifact**.

### 1. Unifying Tools and Humans

If the engine handles network-based tool calls internally, it still requires a completely different, asynchronous mechanism to handle Human-in-the-Loop (since humans respond via UIs, not synchronous network protocols).

By yielding to the host, OpenTeam treats a human exactly like a very slow tool. Mechanically, the engine only needs to understand one concept: **Suspension**.

*   **Tool Call:** Engine hits `query_db` -> Yields to host -> Host runs DB query -> Host resumes engine.
*   **Human Input:** Engine hits `ask_member` (for a Human) -> Yields to host -> Host waits for UI input -> Host resumes engine.

### 2. Preserving the "Portable Artifact" Superpower

Consider a tool call that takes 20 minutes (e.g., triggering a CI build or waiting for a massive database migration). 

If the engine executes tools internally, it must block the thread and hold memory for 20 minutes. 

Because OpenTeam yields, it simply writes `status = suspended_waiting_for_tool` to the SQLite file and terminates the process. The host application can persist the SQLite file to S3 or a database. Twenty minutes later, when a webhook fires, a completely different server can download the SQLite file, inject the tool result, and resume the engine. This makes OpenTeam inherently serverless-friendly and horizontally scalable.

### 3. Keeping the Core Pure and Secure

If OpenTeam handles tool execution internally, the framework suddenly takes on the burden of network topologies, VPC configurations, secrets management (API keys), and retry logic.

By yielding to the host, OpenTeam remains a pure, deterministic state machine. The host application—which already manages the company's secrets, database connections, and network security—takes on the messy reality of execution.

## The Developer's Responsibility

Under this model, integrating with external systems (like internal APIs, custom databases, or third-party protocols) is entirely the responsibility of the developer building the host application. 

The developer writes the execution loop for their application. When they invoke the OpenTeam engine, it will run until it either finishes the task or hits a boundary requiring external input. If it yields for a tool call, the developer's code executes that tool and injects the result back into the engine. If it yields for human input, the developer's code pauses, waits for the user to interact with their UI, and then injects that response back into the engine.

This keeps the OpenTeam framework lean, focused purely on organizational routing and context management, while giving the developer total control over how actions are actually performed in their infrastructure.