# Post-Runtime: From Sidecar to Embeddable Library

This document records an architectural pivot. It explains why the initial gRPC sidecar direction was abandoned, what OpenTeam is becoming instead, and how the project is staged from a stable Go reference implementation to a future Zig embeddable core.

For the original sidecar vision, see [10_runtime.md](./10_runtime.md).

---

## 1. The Initial Decision: gRPC Sidecar as the Engine

The first post-prototype direction treated OpenTeam as a **distributed agentic runtime** — a long-lived daemon deployed as a sidecar alongside the host application.

The design (summarized in `10_runtime.md`) had these properties:

* **Stateless sidecar, stateful artifact.** The Go process held no durable memory. All runs, steps, messages, and suspensions lived in a SQLite file owned by the consumer.
* **gRPC bidirectional stream.** The host and sidecar communicated over a high-performance bus. Tool calls, human-in-the-loop (HITL), and lifecycle hooks (`BeforeToolCall`, `BeforeLLMCall`) were normalized into a single primitive: **Execution Suspension**.
* **Non-blocking reconciliation loop.** When a track hit a suspension, the sidecar wrote state to SQLite, terminated the goroutine, and kept spinning — picking up other pending work while the host resolved I/O out-of-band.
* **Fire-and-forget notifications.** Informational events (`OnLLMStart`, `OnToolComplete`, streaming chunks) streamed to the host without blocking the engine.

This was a serious design. It borrowed from effect-system patterns (ZIO, Cats Effect) and aimed at maximum throughput: a 20-minute human delay or slow API call would consume zero engine CPU.

---

## 2. The Complexity and the Divergence

As the design was pressure-tested against OpenTeam's actual thesis, the sidecar model accumulated **infrastructure weight** that did not serve the core idea.

### Operational and architectural complexity

* **Network as the primary interface.** Every Ask, Act, LLM turn, and observation crossed a process boundary. That introduced serialization, connection management, session lifecycle, backpressure tuning, and failure modes unrelated to context engineering.
* **Multitenancy on a shared daemon.** A sidecar serving many tenants requires isolation, concurrent session limits, artifact routing, noisy-neighbor control, and horizontal scaling decisions — all of which are **host-app problems** dressed up as sidecar problems.
* **Streaming over the wire.** LLM token streaming, CDC events, loop reports, and suspension round-trips competed for bandwidth and ordering guarantees on the same gRPC stream.
* **Two half-abstractions.** The sidecar still needed to understand LLM providers (params, streaming, retries) while also yielding tools and humans to the host. Policy and execution were split awkwardly across the boundary.
* **Deployment tax.** Every consumer needed to run, monitor, version, and secure a separate service — even when the host application already owned all external I/O.

### Divergence from the main goal

OpenTeam's thesis (see [1_idea.md](./1_idea.md)) is not "build another agent server." It is:

> Organize intelligence through teams, channels, and context boundaries — making context engineering a first-class primitive.

The hard problem is **who sees what**, not **how to route HTTP at scale**. The sidecar model reframed OpenTeam as a distributed runtime OS for agents. That is a valid product category, but it is not the product OpenTeam set out to be.

The SQLite artifact, the run tree, mention/liaison semantics, and role-scoped context derivation are the moat. A gRPC sidecar wrapped those ideas in a heavy operational shell and pulled engineering focus toward scheduler tuning, streaming protocols, and sidecar SRE — work that does not make the organizational model clearer or more correct.

---

## 3. The Redesign: Go Library, Yield Everything to Host

The replacement direction is deliberately thin.

### What the engine is

OpenTeam becomes an **embeddable context and orchestration engine** — a library the host links in-process, not a daemon it talks to over the network.

The engine:

* Loads and mutates the **SQLite artifact** (schema, migrations, runs, steps, messages).
* **Derives context** according to OpenTeam rules: channels, roles, tasks, visibility, mention vs. liaison boundaries.
* Advances the **run/step state machine** until it reaches quiescence or a **suspension**.
* Emits **discrete events** (CDC, loop reports, errors) for observation.

The engine does **not**:

* Make HTTP calls (LLM, tools, or otherwise).
* Hold API keys or manage token pools.
* Stream tokens (the only streamable operation was LLM output; that now lives entirely in the host).
* Require a sidecar, gRPC, or separate deployment unit.

### Yield everything to host

Every external effect is a **suspension** resolved by the host:

| Suspension | Host resolves with | Engine continues after |
|------------|-------------------|------------------------|
| **LLM** | Full provider response (or injected mock) | `ResolveLLM` — parse plan, persist, advance |
| **Tool** | Tool execution result | `Act` — same as today |
| **Human** (future) | UI input | Human reply command |

LLM, tools, and humans share one mechanical pattern: the engine writes a pending requirement to SQLite, stops, and waits for the host to inject the result. The host owns rate limits, secrets, retries, streaming UI, and custom before-logic. The engine owns organizational truth.

This extends the yield-to-host decision in [8_tools_and_humans_in_the_loop.md](./8_tools_and_humans_in_the_loop.md) to LLM calls as well — completing the "thin engine" model.

### Go-first API shape (immediate follow-up)

The stable Go reference implementation exposes two complementary interaction modes:

**Push (session loop)**

* `Run(commands, events)` — bidirectional channels.
* Host sends **commands**: `Ask`, `Act`, `ResolveLLM`.
* Engine sends **events**: CDC (mention, action, message), loop reports, errors.
* While `Run` is active, all writes go through the command channel (single writer).

**Pull (reconciliation)**

* `Ask`, `Act` — direct pull methods for tests and simple flows.
* `Frontier()` / `Snapshot()` — read SQLite when the host missed events or reconnects after a crash.

Events are hints. **The artifact is authoritative.** If the host drops an event, it catches up from the database.

A thin **`host` package** (Go SDK) implements the suspension loop: run until suspended, resolve externally, resume. This package is the prototype for future FFI callbacks in other languages.

---

## 4. Benefits of the Library Model

### Alignment with the thesis

Engineering effort returns to context boundaries, run-tree semantics, and the mention/liaison model — the problems OpenTeam exists to solve.

### SQLite-like product identity

The artifact file is already the portable contract. An embeddable library completes the picture: same as SQLite, the host loads a file, calls into the library, and persists the result. No network protocol to version.

### Multitenancy without a shared daemon

Each tenant or conversation gets its own artifact in the host's worker process. Isolation follows the application's existing scaling model. No sidecar pool to tune, no cross-tenant contention on a central scheduler.

### Simpler surface area

* No gRPC schema, no chunk streaming protocol, no sidecar deployment.
* Discrete suspensions and events — not a firehose of tokens over the wire.
* Single-writer SQLite per artifact; concurrency is the host's problem at the worker level.

### Security and operability

* No API keys in the engine process.
* No outbound network from the library (air-gap friendly).
* Easier to audit: pure state machine + SQL.

### Clear path to other languages

Go proves the semantics. The stable artifact schema and C ABI (future) become the portable contract — language SDKs wrap the same suspensions and commands, not a bespoke gRPC dialect per release.

---

## 5. Final Goal: Port to Zig After Go Is Stable

Go is the **reference implementation**, not the final form factor.

### Why Go first

* The working prototype, tests, and sqlc queries already exist in Go.
* Community adoption via `go get` is the fastest path to battle-test the organizational model.
* Semantics can iterate without simultaneously solving FFI, sqlc-gen-zig beta, and C ABI memory ownership.

### What must be frozen before the Zig port

These are the real API contracts — not Go types or gRPC protos:

1. **SQLite schema and migrations** (SQL-first, versioned artifact format).
2. **Suspension kinds and command shapes** (Ask, Act, ResolveLLM).
3. **Context derivation rules** (what enters a turn, visibility, channel boundaries).
4. **Run tree and join semantics** (parent/child runs, mention joins).
5. **Golden artifact tests** — fixture `.db` in, expected `.db` out; language-agnostic proof of correctness.

### Why Zig

Zig is the pragmatic choice for the embeddable core:

* Stable **C ABI** without a GC or runtime in the `.so` (unlike Go `-buildmode=c-shared`).
* Small binaries, suitable for sidecar-less embedding in Zig, C, Python, Rust, and other hosts via FFI.
* Same SQL files and sqlc-gen-zig (community plugin) can codegen queries against the frozen schema.
* SQLite via the C API (`sqlite3.h` or `libsql.h` through `@cImport`) — no native Turso/Zig driver required.

The Go engine remains valuable as a reference and test oracle even after the Zig library ships.

### Staged roadmap

```text
Phase 1 (now)     Go library — yield everything, no HTTP in engine,
                  Run + commands + Frontier, host SDK, golden DB tests

Phase 2           Community adoption — validate channels, roles, suspensions,
                  and artifact compatibility in real applications

Phase 3           Zig embeddable core — same schema, same behavior, C ABI,
                  language SDKs wrap callbacks instead of gRPC
```

gRPC sidecar may reappear later as an **optional wrapper** over the C API for hosts that cannot embed a library. It is not the center of gravity.

---

## 6. In One Sentence

OpenTeam is not a distributed agent runtime — it is an **organizational context engine** you embed like SQLite, yielding every external effect to the host, proven first in Go and ported to Zig once the idea is stable.
