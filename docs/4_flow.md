# Data Flow & Execution Model

OpenTeam does not use rigid DAGs (Directed Acyclic Graphs) to route information. Instead, it relies on an event-driven data flow powered by the LLMs themselves.

The framework gives agents the "steering wheel" to navigate the organization, but strictly bounds where they can steer based on their assigned Roles and Channels.

## 1. The Tool Taxonomy

To achieve this, OpenTeam splits tools into two distinct categories:

### A. Expert Tools (Domain Logic)
These are the standard tools developers write to interact with the outside world.
*   *Purpose:* To gather data or execute actions (e.g., query a database, search a codebase, check an API).
*   *Scope:* They return data directly back to the agent's current Duty. They **do not** change the state of the conversation or route messages to other participants.
*   *Concurrency:* Agents can invoke multiple Expert Tools in parallel to gather data simultaneously.

### B. Protocol Tools (Control Flow)
These are built-in tools provided by the OpenTeam framework. They dictate *how* the agent interacts with the organization. Instead of writing complex `if/else` routing logic in code, the developer gives the agent Protocol Tools to manage the flow.

There are three primary types of Protocol Tools:

1.  **The Duty-Pass Tool:** Used when an agent successfully completes a preliminary Duty (like a security check) and wants to pass the context to its next Duty in the sequence.
2.  **The Ask-Member Tool:** Used to explicitly invoke one or more other Members *who are in the same Channel*. This allows for intra-channel collaboration and can trigger parallel agent executions if multiple members are asked.
3.  **The Liaison Tool:** Used to bridge boundaries. An agent acting as a Liaison uses this tool to summarize the current context and post it into a *different* Channel.

## 2. The Duty Pipeline (Intra-Agent Flow)

When a new message arrives in a Channel, the framework wakes up the relevant Member. The data flows through that Member's assigned Duties sequentially.

This acts exactly like "Guard Clauses" in traditional programming.

**Step 1: The Guardrail Duty (e.g., Security/Relevance)**
*   The LLM is invoked with the first Duty's instructions.
*   *Short-circuit:* If the LLM detects an issue (e.g., a jailbreak or an off-topic request), it ignores the Protocol Tools and simply outputs a text response. The turn ends immediately, protecting the rest of the system.
*   *Pass:* If the LLM determines it is safe, it calls the **Duty-Pass Tool**.

**Step 2: The Execution Duty (e.g., The Worker)**
*   The framework receives the Duty-Pass call and loads the next Duty.
*   The LLM is invoked with the Worker instructions.
*   It may call **Expert Tools** to gather data.
*   Once it has the data, it decides how to proceed:
    *   Reply directly to the Channel.
    *   Call the **Ask-Member Tool** to get help from a peer in the room.
    *   Call the **Liaison Tool** to escalate the issue to another channel.

## 3. The Context Boundary Enforcement

The distinction between the "Ask-Member" tool and the "Liaison" tool is what enforces the context boundaries.

*   If Alice needs Bob's help, and they are both in the `#dev-channel`, she uses the **Ask-Member Tool**. The conversation history remains isolated in the `#dev-channel` channel.
*   If Alice needs to tell the Client what happened, she *cannot* use the Ask-Member tool, because the Client is not in the `#dev-channel`. She must use the **Liaison Tool** to drop a clean summary into the `#client-support`.

This guarantees that messy internal reasoning (logs, arguments, tool outputs) never accidentally leaks across Channel boundaries.