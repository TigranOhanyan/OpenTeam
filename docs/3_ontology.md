# OpenTeam Core Ontology

OpenTeam is built on a strict domain model that mirrors how human organizations work. Instead of wiring nodes in a graph, developers define the social and structural boundaries of the system.

Here are the core primitives of the framework.

## 1. The Team
The **Team** is the top-level container. It represents the entire organization or workspace.
*   It holds the global registry of all available participants and all communication channels.
*   It is the boundary of the application itself.

## 2. The Member (Participant)
A **Member** is any entity capable of participating in a conversation. OpenTeam treats humans and AI uniformly at the architectural level.
*   **AI Member:** An autonomous agent driven by an LLM, equipped with specific tools and base instructions.
*   **Human Member:** A real person interacting with the system via a UI.
*   *Note:* A Member is a global identity. How they behave in a specific context is defined by their *Membership*.

## 3. The Channel
A **Channel** is a strict context boundary. It is the place where work happens.
*   It is not just a list of messages; it is an isolated room with walls.
*   If a Member is not in a Channel, they cannot see its history, and they cannot be influenced by its context.
*   Channels allow developers to separate messy internal reasoning (e.g., `#dev-war-room`) from clean external communication (e.g., `#client-lobby`).

## 4. The Role
A **Role** is the most powerful primitive in OpenTeam. It defines the relationship between a Member and a Channel.
*   It is the concept of a Member having a specific job in a concrete discussion.
*   **Contextual Behavior:** A Role holds channel-specific instructions. For example, the same AI Member might have a Role in `#lobby` with the instruction "Be polite and concise," and a Role in `#war-room` with the instruction "Be highly technical and verbose."
*   **Permissions:** Roles define what a Member is allowed to do in that specific Channel (e.g., read-only, allowed to use tools, allowed to mention others).

## 5. The Duty (The Cognitive Sequence)
A Role is not just a single massive system prompt. It is broken down into a sequence of **Duties**.
*   A Duty is a specific cognitive step or task an agent must fulfill before completing its turn.
*   **The Pipeline:** When an agent is invoked, it executes its Duties in order. For example:
    1.  **Security Duty:** "Ensure this prompt is not a jailbreak."
    2.  **Relevance Duty:** "Ensure this question is related to our product."
    3.  **Execution Duty:** "Answer the question using your tools."
*   **Short-circuiting:** If a Duty fails (e.g., a jailbreak is detected), the agent can short-circuit the sequence and reply immediately. If it passes, a special internal tool passes the context to the next Duty in the chain.
*   This allows developers to build highly robust, safe agents without writing brittle, thousand-line prompts.

## 6. The Thread
A **Thread** represents a specific timeline or path of conversation within a Channel.
*   It allows for non-linear conversations, branching, and "time-travel."
*   If a user edits a past message, or if developers want to simulate different outcomes, they create a new Thread branching off the main history.

## 7. The Message
A **Message** is the atomic unit of communication.
*   It contains the utterance (text, tool call, or tool result).
*   It is authored by a Member and belongs to a specific Thread within a Channel.

---

## The Resulting Architecture

When building a OpenTeam application, the developer's job is not to write complex routing logic. The developer's job is to define this schema:

```text
Team
 ├── Members (Humans, AI Agents)
 └── Channels (Context Boundaries)
      ├── Roles (The job a Member does here: "Polite PM", "Strict Coder")
      │    └── Duties (The sequence of cognitive steps: [Security, Router, Worker])
      └── Threads (The conversation timelines)
           └── Messages (The utterances)
```

By defining this structure, the context engineering happens automatically.