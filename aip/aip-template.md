---
AIP-ID: <id>
Title: <title>
Status: READY | DRAFT | IN_PROGRESS | BLOCKED | REVIEW_REQUIRED | COMPLETED
Version: 1.0
Last-Updated: <yyyy-mm-dd>
---

# Goal
<One sentence describing the final, desired outcome.>

# Context
<Brief but essential background. Include links to relevant code, architecture diagrams, or prior decisions.>

# Phased Plan
- [ ] Phase 1: <Phase Name>
  - [ ] 1.1: <Task 1>
  - [ ] 1.2: <Task 2>
- [ ] Phase 2: <Phase Name>
  - [ ] 2.1: <Task 1>
  - [ ] 2.2: <Task 2>
- [ ] Phase 3: <Phase Name>
  - [ ] 3.1: <Task 1>

# Constraints & Guardrails
- Language: <e.g., Go 1.2x only>
- Dependencies: <e.g., internal bcrypt package only>
- Performance: <e.g., P99 latency <150ms>
- Style: <e.g., formatted with gofmt before commit>
- State: <rules for files, commits, etc.>

# Definition of Done (DoD)
- [ ] All items in the Phased Plan are complete.
- [ ] All constraints are met.
- [ ] Service deployed to staging environment.
- [ ] CI/CD pipeline passes successfully.

# Maintenance Protocol

## Execution Loop
1. Read: Parse AIP (goal, state, next task).
2. Execute: Perform the first unchecked task.
3. Verify: Confirm task success.
4. Update: Mark task complete and update Last-Updated timestamp.
5. Repeat until completion.

## State Management
- Before work: set Status → IN_PROGRESS.
- After last DoD item: set Status → COMPLETED.

## Handling Blockers
- Stop work and set Status → BLOCKED.
- Add comment under header:  
  `# BLOCKER: <reason>`
- Wait for human intervention.

## Proposing Plan Deviations
- Stop work and set Status → REVIEW_REQUIRED.
- Add a PROPOSAL block with explanation and justification.
- Await human approval; if approved, status resets to READY.
