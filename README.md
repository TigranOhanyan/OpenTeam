# OpenTeam

> ⚠️ **WARNING:** This project is currently in active development. APIs, structures, and features are subject to change and are **not backward compatible**.

## What is OpenTeam?

Most agent frameworks treat intelligence as nodes in a graph, chains of calls, tool pipelines, or state machines. While useful for execution, these abstractions often fail to address the real difficulty of building capable agentic systems: **context engineering**. 

The hard problem is deciding who sees what, who is allowed to speak, and which context is private versus shared. It is about how specialists collaborate without polluting each other's reasoning, and how messy internal work is translated into clean external communication.

**OpenTeam** is a framework for organizing intelligence through teams, channels, and context boundaries, making context engineering a first-class primitive instead of an afterthought.

It treats an agentic system as an organization made of teams, channels, and communication boundaries—similar to how people organize real work in tools like Slack or Microsoft Teams. 

Instead of asking only, "What is the next step in the workflow?", OpenTeam asks, "What is the right organizational structure for this intelligence?"

## The Metaphor

To understand how OpenTeam works, look at how a modern software team operates:
- **Members (Participants):** Agents or humans with specific roles (e.g., Client, SRE, PM, Backend Dev).
- **Channels (Context Boundaries):** Public or private spaces where specific types of communication happen (e.g., `#client-support`, `#triage`, `#backend-dev`).
- **Flow (Progressive Context Refinement):** Information is translated and refined as it moves between channels. The Backend Dev doesn't need to see the Client's raw complaints, and the Client doesn't need to see the database logs.

OpenTeam allows you to build this exact structure in code. You define the channels, assign members their roles, and let them collaborate effectively without context pollution.

## Documentation

For more details on the concepts, architecture, and implementation details behind OpenTeam, please refer to the [docs](./docs) directory.
