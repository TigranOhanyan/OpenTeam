# The Metaphor: The Software Team

To understand how OpenTeam works, you don't need to learn a new paradigm. You just need to look at how a modern software team operates in Slack or Teams.

Imagine a typical software agency. A client finds a bug. How does the fix actually get shipped?

## The Setup

The agency doesn't just put everyone in one giant chat channel. That would be chaos. Instead, they organize by **Channels** and **Roles**.

### The Members (The Participants)
*   **Dave (The Client):** The user experiencing the problem.
*   **Alex (The SRE):** The frontline monitor. He watches the dashboards and does the initial technical translation.
*   **Sarah (The PM):** The router. She manages the backlog, decides what gets worked on, and assigns tasks.
*   **Marcus (The Backend Dev):** The specialist. He writes the code and touches the database.

### The Channels (The Context Boundaries)
*   **`#client-support`:** A public channel. Only Dave and Alex are here. The tone is polite and focused on the user experience.
*   **`#triage`:** An internal channel. Alex and Sarah are here. This is where bugs are translated into tickets and prioritized.
*   **`#backend-dev`:** A private execution channel. Sarah and Marcus are here. This is where the actual code is written.

## The Flow: Progressive Context Refinement

It is 3:00 PM on a Tuesday.

### 1. The Intake (`#client-support`)
Dave posts in `#client-support`: *"Hey, the checkout button is spinning forever. Customers can't buy anything."*

Alex (the SRE) reads this. He doesn't immediately wake up the whole engineering team. He replies to Dave, gets the session ID, and checks the Datadog dashboards. He sees a massive spike in database latency.

### 2. The First Escalation (`#triage`)
Alex acts as a **Bridge**. He switches to the internal `#triage` channel to talk to Sarah.

He leaves the polite customer-service tone behind and drops the technical context:
*"Hey @Sarah, Dave is reporting checkout hangs. I checked the graphs and we have a massive latency spike on the `orders` table. This isn't a frontend glitch; it's a backend bottleneck."*

Sarah reads it. She knows the team is in the middle of a sprint, but a checkout bug is a Sev-1. She makes the routing decision: this requires the backend team immediately.

### 3. The Second Escalation (`#backend-dev`)
Sarah acts as the next **Bridge**. She switches to `#backend-dev` and wakes up the specialist.

*"@Marcus, dropping a Sev-1 on you. Checkout is hanging due to latency on the `orders` table. Alex has the session ID in triage. Can you look at this immediately?"*

Marcus wakes up. Notice what Marcus *doesn't* have to do:
* He doesn't have to talk to Dave to calm him down.
* He doesn't have to look at the Datadog graphs that Alex already checked.
* He doesn't have to decide if this is more important than his current sprint work (Sarah already decided that).

Marcus just receives a highly refined, perfectly scoped technical task. He pulls the database logs, finds a missing index from yesterday's migration, and pushes a hotfix.

### 4. The Roll-up
The fix flows back up the chain.

Marcus replies in `#backend-dev`: *"Fix merged. Index added. Latency is dropping."*

Sarah goes back to `#triage`: *"Marcus pushed the fix. Alex, can you confirm the graphs look good?"*

Alex confirms the graphs are green in `#triage`, then switches back to `#client-support`. He translates the technical fix back into a clean, client-friendly summary.

*"Hi Dave, we found a database configuration issue that was causing the timeout. We've deployed a hotfix and the checkout is processing normally again."*

***

## How This Maps to OpenTeam

When we build AI systems, we usually try to build one massive brain that can talk to Dave, read the Datadog graphs, prioritize the backlog, and write the SQL fix all at once. When it gets confused by all that conflicting context, we try to fix it by writing a longer prompt.

OpenTeam argues that the solution isn't a bigger brain. The solution is the organizational structure that this team used.

*   **The Members:** Dave, Alex, Sarah, and Marcus are the agents (or humans) in the system.
*   **The Channels:** `#client-support`, `#triage`, and `#backend-dev` are strict context boundaries.
*   **The Roles & Duties:** Alex has the duty of translating user complaints into technical metrics. Sarah has the duty of routing. Marcus has the duty of writing code.
*   **The Mentions:** Sarah doesn't write a hardcoded script to invoke Marcus. She just mentions `@Marcus` in the channel, and he wakes up.
*   **Context Isolation:** Dave never sees Marcus's SQL logs. Marcus never sees Dave's complaints. The context is progressively refined as it moves through the channels.

OpenTeam is a framework that lets you build this exact structure in code. You define the channels, you assign the members their roles, and you let them work.