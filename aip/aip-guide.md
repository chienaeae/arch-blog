# Guide: The Actionable Implementation Plan (AIP) Protocol

This document defines the purpose, structure, and maintenance protocol for an **Actionable Implementation Plan (AIP)**.  
All agents assigned to a development task must adhere to this protocol to ensure **consistency, state persistence, and predictable outcomes**.

---

## 1. What is an AIP?

An **Actionable Implementation Plan (AIP)** is a **live, machine-readable document** that serves as the **single source of truth** for a specific development task, such as implementing a feature, executing a refactor, or fixing a complex bug.

- Unlike a **Request for Comments (RFC)** which is static and for human discussion,  
  an **AIP is a dynamic execution script**.  
- Its primary audience is an **AI agent**.

### Core Principles

- **Actionable**: Every part of the plan is designed as an executable instruction.  
- **Stateful**: The AIP reflects the real-time status of the task using checklists and status flags.  
- **Single Source of Truth**: The AIP is the definitive guide for the task.  
  - Any change in direction must be reflected in the AIP before execution.

---

## 2. How to Write an AIP

An AIP must be a **single Markdown file (`.md`)** and follow this structure exactly.

### Header

The file begins with a **YAML-like header** containing essential metadata:

```yaml
AIP-ID: 001
Title: Refactor User Authentication Service to Go
Status: READY | DRAFT | IN_PROGRESS | BLOCKED | REVIEW_REQUIRED | COMPLETED
Version: 1.0
Last-Updated: 2025-08-17
```

**Field Definitions:**
- **AIP-ID**: A unique identifier.  
- **Title**: A concise description of the task.  
- **Status**: Must be one of the specified states.  
- **Version**: Follows semantic versioning (e.g., `1.0`, `1.1`).  
- **Last-Updated**: Timestamp of the last modification.  

---

### Sections

#### 1. Goal
A one-sentence description of the **desired outcome**.  
**Example**:  
*To replace the legacy Python authentication service with a new, high-performance Go service without any downtime.*

#### 2. Context
Brief background, including links to code, diagrams, or prior decisions.  
**Example**:  
*The current service in `/services/auth-python` suffers from high latency. The new service will use shared Go libraries in `/pkg/database`.*

#### 3. Phased Plan
A checklist of **discrete, verifiable steps**. Use nested lists for granularity.  

**Example:**

```markdown
- [ ] Phase 1: Project Scaffolding
  - [ ] 1.1: Create `/services/auth-go` directory.
  - [ ] 1.2: Initialize Go module: `go mod init <module_name>`.
  - [ ] 1.3: Add dependencies (Gin for routing, GORM for DB).
- [ ] Phase 2: Implement Core Logic
  - [ ] 2.1: Define `User` struct from `/schemas/user.sql`.
  - [ ] 2.2: Implement `Login(username, password)` function.
- [ ] Phase 3: Testing & Validation
  - [ ] 3.1: Write unit tests for `Login`, >90% coverage.
```

#### 4. Constraints & Guardrails
Strict rules the agent **must not violate**.  

**Example:**
- **Language**: Go 1.2x only.  
- **Dependencies**: Use internal bcrypt in `/pkg/security`.  
- **Performance**: P99 latency < 150ms for Login endpoint.  
- **Style**: Code formatted with `gofmt` before commit.  
- **State**: Do not commit generated files or vendor directories.  

#### 5. Definition of Done (DoD)
Objective criteria proving task completion.  

**Example:**

```markdown
- [ ] All items in the Phased Plan are complete.
- [ ] All constraints are met.
- [ ] Service deployed to staging.
- [ ] CI/CD pipeline passes successfully.
```

---

## 3. How to Maintain an AIP

Rules for agents interacting with an AIP.

### 1. The Execution Loop
1. **Read**: Parse AIP (goal, state, next task).  
2. **Execute**: Perform the first unchecked task.  
3. **Verify**: Confirm success.  
4. **Update**:  
   - Mark task as complete (`- [x]`).  
   - Update `Last-Updated` timestamp.  
5. **Repeat** until all tasks are done.  

---

### 2. State Management
- Before work: set `Status` → `IN_PROGRESS`.  
- After last DoD item: set `Status` → `COMPLETED`.  

---

### 3. Handling Blockers
If blocked by error/ambiguity/external factor:  
1. Stop work.  
2. Change `Status` → `BLOCKED`.  
3. Add comment under header:  

```markdown
# BLOCKER: Database credentials for staging are invalid.
```

4. Wait for human intervention.  

---

### 4. Proposing Plan Deviations
If a better path conflicts with current plan:  
1. Stop work.  
2. Change `Status` → `REVIEW_REQUIRED`.  
3. Add a `PROPOSAL:` block explaining change + justification.  

**Example:**

```markdown
PROPOSAL: Use sqlx instead of GORM to reduce complexity in Phase 2.
```

4. Wait for human approval → then status returns to `READY`.  
