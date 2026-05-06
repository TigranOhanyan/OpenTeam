# The Idea

## The Question

How should intelligence be organized?

Most agent frameworks answer this question with software-centric abstractions. They treat intelligence as nodes in a graph, chains of calls, tool pipelines, or state machines with prompts attached.

These abstractions are useful for execution, but they do not match the real difficulty of building capable agentic systems.

## The Problem

The hard problem is not only deciding what runs next. 

The hard problem is deciding who sees what, who is allowed to speak, and which context is private versus shared. It is about how specialists collaborate without polluting each other's reasoning, and how messy internal work is translated into clean external communication.

In practice, this is a context engineering problem.

Existing frameworks often treat context engineering as a secondary concern. Developers end up injecting callbacks, mutating prompts, filtering histories, or building custom routing logic around abstractions that were never designed for visibility and boundaries.

As a result, the most important part of the system is often implemented as a workaround.

## The Core Insight

Context engineering should not be a hidden trick inside the framework. It should be the framework.

OpenTeam starts from a different premise: intelligence should be organized the way humans organize real work. Not as nodes in a graph, but as people in an organization.

## The Idea

OpenTeam treats an agentic system as an organization made of teams, channels, and communication boundaries.

The central abstraction is not the graph. The central abstraction is the social structure in which intelligence operates. This includes teams, channels, private discussions, escalations, handoffs, and summaries between contexts.

This is similar to how people already think about Slack or Teams. Some conversations are public, while others are private. Some members are always in the channel, while others are only brought in when needed. Specialists work in focused spaces, and someone is responsible for translating that internal work into an external answer.

This model is intuitive, but it is not only a metaphor. It is a practical, programmable way to structure machine intelligence.

## Why This Matters

LLM systems often fail for organizational reasons. They fail because they are given too much irrelevant context, their roles blur, or there is no boundary between internal reasoning and user-facing communication. When every agent reads the same massive transcript, or when handoffs between specialists are ad hoc, the system breaks down.

OpenTeam treats these failures as structural problems, not prompt-writing problems.

Instead of asking only, "What is the next step in the workflow?", OpenTeam asks, "What is the right organizational structure for this intelligence?"

That is the core idea.

## In One Sentence

OpenTeam is a framework for organizing intelligence through teams, channels, and context boundaries, making context engineering a first-class primitive instead of an afterthought.
