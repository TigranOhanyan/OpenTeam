# OpenTeam Framework Evolution (Iteration 1)

This document outlines the architectural transition of OpenTeam from a recursive, call-stack-driven engine into an **event-sourced, append-only DAG (Directed Acyclic Graph) state machine**. 

For this first iteration, we are focusing on schema upgrades, the unification of the cognitive loop, and a synchronous event-loop implementation, while intentionally keeping the public API (`Ask` and `Run`) unchanged.

---

## 1. The Core Paradigm Shift

1. **Unified Cognitive Loop (`Step`)**: We are removing the artificial separation of `observing`, `reasoning`, and `acting`. A `Step` is now a single, cohesive execution node that represents one full ReAct cycle.
2. **Append-Only Immutability**: Steps never go backwards. A `Step` transitions from `PENDING` -> `RUNNING` -> `COMPLETED`. If an agent needs to wait and resume, it doesn't rewind its current Step; instead, a *new* `Step` is appended to its `Run`.
3. **No More Call-Stack Recursion**: Agents no longer recursively call functions to spawn sub-agents. They write `PENDING` state to the database and terminate. A central event loop picks up the work.

---

## 2. Updated Schema and Relations

The system separates the **Social Layer** (communication) from the **Execution Layer** (orchestration).

### The Social Layer

#### `messages`
The `Message` is the causal root of the tree. It represents a fact of communication. 
* **Changes**: `step_id` and `task_id` become **NULLABLE**.
* **Logic**: If a human writes the message, `step_id` and `task_id` are `NULL` (humans do not execute steps or tasks). If an agent writes the message, they are populated with the `Step` that authored it.

#### `mentions`
A `Mention` is a call-to-action embedded inside a `Message`.
* **Relations**: Belongs to a `message_id`. Targets a specific `member_name`.

### The Execution Layer

#### `runs`
A `Run` is the state-tracking container for an agent completing an assignment.
* **Relations**: Points to the `mention_id` that triggered it. (We drop `source_step_id` because the explicit DAG edges are now in `run_links`, and provenance is traced through Mention -> Message).
* **State**: Tracks `status` (`PENDING`, `RUNNING`, `SUSPENDED`, `COMPLETED`). 

#### `steps` (The Unified Execution Node)
A `Step` is a single iteration of the ReAct loop.
* **Relations**: Belongs to a `run_id`.
* **State**: Tracks `status` (`PENDING`, `RUNNING`, `COMPLETED`).
* **Cleanup**: `step_links` table is deleted. Chronological order is guaranteed by ULID sorting. Dummy "asking" and "observing" steps are deleted.

#### `run_links` (The Explicit DAG Edges)
An explicit link table to isolate execution graph logic from social messages.
* **Schema**: `parent_run_id` (parent), `spawning_step_id` (the exact step that spawned the child), `child_run_id` (child).
* **Purpose**: Allows the engine to traverse, suspend, and wake up runs natively without joining across the Social layer tables.

---

## 3. The Public API (Intermediate Phase)

We retain the explicit `Ask` and `Run` methods for the host application, but their internal mechanics shift to database-driven queues.

### `Ask(message)`
1. Writes the user's `Message` (`step_id = NULL`).
2. Parses and writes the `Mentions` linked to that `Message`.
3. Spawns a `Run` for each mention with `status = PENDING`.
4. Creates an initial `Step` for each Run with `status = PENDING`.
5. **Returns**: The `message_id` (No longer a dummy step ID).

### `Run(messageId)`
1. Looks up the original `Message` and finds all the `Runs` it initially spawned.
2. Starts the **Synchronous Single-Threaded Event Loop** (see below).

---

## 4. The Synchronous Event Loop (Controller Pattern)

Inside `Run(messageId)`, the system executes a synchronous loop in the current goroutine. This loop acts as a true controller, decoupling the execution of work from the orchestration of the DAG.

```go
func Run() {
    for {
        // Phase 1: Reconciliation (The Wake-Up Check)
        // Find all SUSPENDED runs and their children, evaluate readiness in Go
        readyRuns := FindSuspendedRunsReadyToResume()
        for _, run := range readyRuns {
            run.Status = "PENDING"
            InsertNewPendingStep(run.ID) // Append a new cognitive loop
        }

        // Phase 2: Execution (The Worker)
        // Find the next PENDING step
        pendingStep := FindNextPendingStep()
        
        if pendingStep == nil {
            // If no steps are pending, and no runs are suspended waiting on children,
            // the entire execution tree is fully resolved.
            if !AreAnyRunsSuspended() {
                break 
            }
            continue 
        }
        
        // The worker executes the step (Observe -> Reason -> Act).
        ExecuteUnifiedStep(pendingStep)
    }
}
```

---

## 5. Backtracking (Pull-Based Reconciliation)

We use a **Pull-Based / Controller** approach for backtracking. Individual workers (`ExecuteUnifiedStep`) are "dumb"—they only ever modify their own state and never check on their siblings or wake up their parents.

All backtracking logic happens centrally in **Phase 1** of the Event Loop using the `run_links` table.

The engine runs a query to pull all `SUSPENDED` runs and their child run statuses into memory, allowing the Go code to handle the business logic of determining readiness:

```sql
-- Find all SUSPENDED runs and the status of any children they spawned
SELECT 
    r.id AS parent_run_id,
    child.id AS child_run_id,
    child.status AS child_status
FROM runs r
LEFT JOIN run_links rl ON r.id = rl.parent_run_id
LEFT JOIN runs child ON rl.child_run_id = child.id
WHERE r.status = 'SUSPENDED';
```

**Go Logic**:
1. Group the results by `parent_run_id`.
2. For each parent, check if all `child_status` values are `'COMPLETED'`.
3. If true, the parent is ready. Wake it up by marking it `PENDING` and inserting a new Step.

**Why this is the ultimate design:**
1. **Dumb Workers**: The agent execution logic is completely isolated from the DAG orchestration logic.
2. **Race-Condition Proof**: Because the central loop evaluates the global state *after* workers finish, we eliminate the classic race condition where two children finish simultaneously and double-wake (or never wake) the parent.
3. **Pure Event Sourcing**: The engine is a true stateless controller. It simply observes the database state and pushes it forward.