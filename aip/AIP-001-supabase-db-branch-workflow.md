---
AIP-ID: 001
Title: Migrate from Flyway to Supabase DB Branch Workflow with Schema/Data Responsibility Separation
Status: READY
Version: 1.0
Last-Updated: 2025-08-17
---

# Goal
Replace the manual Flyway-driven database process with an automated, modular, and scalable Supabase-driven workflow that cleanly separates schema (Supabase migrations) from data state (Golang seeders), fully integrated with CI/CD.

# Context
The current repository uses Flyway-style SQL migrations in `db/migrations` (e.g., `V1__create_users_table.sql`, `V2__create_authz_tables.sql`, `V3__create_posts_and_themes_tables.sql`). The backend is a Go service under `backend/` with layered `domain`/`service`/`repository` packages and existing seeder scaffolding (e.g., `internal/authz/seeder`, `platform/seeder`). Supabase will manage schema diffs, versioned migrations, and DB Branches for PRs, while Golang programs will ensure idempotent baseline data and optional test fixtures.

This plan follows the official Supabase migration workflow, including local development (`supabase migration new`, `supabase migration up`, `supabase db diff`, `supabase db reset`) and deployment (`supabase db push`) as documented in [Supabase: Database migrations](https://supabase.com/docs/guides/deployment/database-migrations).

# Phased Plan
- [ ] Phase 1: Migrate from Flyway to Supabase
  - [ ] 1.1: Remove Flyway from CI/CD and developer docs; keep existing SQL files for conversion.
  - [ ] 1.2: Install Supabase CLI locally and in CI; add `supabase/config.toml` and initialize project metadata (`supabase/`).
    - [ ] 1.2.1: Authenticate CLI: `supabase login`.
    - [ ] 1.2.2: Link repository to project: `supabase link` (stores project ref for CLI commands).
  - [ ] 1.3: Create `supabase/migrations` directory; adopt timestamp naming `YYYYMMDDHHMMSS_description.sql`.
  - [ ] 1.4: Convert existing Flyway files in `db/migrations` to Supabase timestamped files and move them to `supabase/migrations/` (preserve semantic order).
    - [ ] 1.4.1: Ensure SQL is compatible with Supabase Postgres; fix any Flyway-specific directives.
    - [ ] 1.4.2: Verify locally using CLI: apply via `supabase migration up` and validate with `supabase db reset` (clean re-apply without errors, correct schema state).
  - [ ] 1.5: Deprecate and remove `db/migrations` after successful conversion; update references.
  - [ ] 1.6: Link repository to Supabase project and enable DB Branches on PRs.
    - [ ] 1.6.1: Configure CI secrets: `SUPABASE_ACCESS_TOKEN`, project `SUPABASE_PROJECT_REF`, and environment URLs/keys.
    - [ ] 1.6.2: Grant least-privilege credentials for CI seeders.
  - [ ] 1.7: Update `justfile` (or makefile/scripts) with tasks: `db:diff`, `db:reset`, `db:migrate`, `db:branch:status` that wrap Supabase CLI.
  - [ ] 1.8: Replace any `flyway migrate` steps in pipelines with Supabase DB Branch workflow.
  - [ ] 1.9: Add optional local seed file `supabase/seed.sql` strictly for developer convenience (local `supabase db reset`), keeping production baseline data in Golang seeders.

- [ ] Phase 2: Establish Standardized Database Management Workflow
  - [ ] 2.1: Schema via Supabase Migrations
    - [ ] 2.1.1: Developers generate migrations using `supabase db diff` against local dev DB; commit SQL under `supabase/migrations/` with descriptive suffixes.
    - [ ] 2.1.2: Require code review for all migration files; forbid edits to previously applied migrations.
    - [ ] 2.1.3: Include RLS policies, indexes, and constraints in migrations alongside tables and functions.
    - [ ] 2.1.4: Document local workflow: start local DB, apply latest migrations, run app.
    - [ ] 2.1.5: Validate migrations locally using:
      - Create new migration: `supabase migration new <name>` then edit the generated SQL.
      - Apply up: `supabase migration up`.
      - Diff dashboard/manual changes: `supabase db diff -f <name>`.
      - Full reset for test: `supabase db reset` (reapplies migrations and optional `supabase/seed.sql`).
      
      Example commands:
      
      ```bash
      supabase migration new create_employees_table
      supabase migration up
      supabase db diff -f add_department_column
      supabase db reset
      ```
      
      Example SQL snippets (in generated migration files):
      
      ```sql
      create table if not exists employees (
        id bigint primary key generated always as identity,
        name text not null,
        email text,
        created_at timestamptz default now()
      );
      ```
      
      ```sql
      alter table if exists public.employees
        add column if not exists department text default 'Hooli';
      ```
    - [ ] 2.1.6: For non-branch remote environments (if applicable), deploy schema via `supabase db push` (optionally `--include-seed` for non-production only).
  - [ ] 2.2: Baseline Data (Idempotent) via Golang Program
    - [ ] 2.2.1: Create `backend/cmd/seed-baseline` (CLI) to initialize core roles/permissions and other critical data via existing `domain/service/repository` layers.
    - [ ] 2.2.2: Implement strict idempotency: safe upserts, versioned seeds, and transactional execution per logical unit.
    - [ ] 2.2.3: Accept DB connection via env/flags (e.g., `DATABASE_URL`); no embedded credentials.
    - [ ] 2.2.4: Add unit/integration tests to verify repeated runs produce no duplicates and correct end state.
    - [ ] 2.2.5: Integrate with CI to run against DB Branch and staging environments.
  - [ ] 2.3: Test Environment Data via Separate Script
    - [ ] 2.3.1: Create `backend/cmd/seed-fixtures` with datasets (e.g., `--dataset smoke|full`).
    - [ ] 2.3.2: Keep fixtures isolated from baseline seed; never run on production.
    - [ ] 2.3.3: Provide easy teardown/reset helpers for developer convenience.

- [ ] Phase 3: CI/CD Integration and End-to-End Automation
  - [ ] 3.1: Local Development
    - [ ] 3.1.1: Developer generates migrations with `supabase db diff`; commits code + migrations in the same branch.
    - [ ] 3.1.2: `just db:migrate` (or equivalent) applies migrations locally for rapid feedback.
  - [ ] 3.2: Pull Request Flow
    - [ ] 3.2.1: On PR open, Supabase creates a DB Branch and applies the branch's migrations.
    - [ ] 3.2.2: CI waits for DB Branch ready, then runs `seed-baseline` against the branch database.
    - [ ] 3.2.3: (Optional) CI runs `seed-fixtures` to populate test data for integration tests.
    - [ ] 3.2.4: Execute application tests against the PR DB Branch; publish results.
  - [ ] 3.3: Merge to Main / Deployment
    - [ ] 3.3.1: Supabase applies migrations to production when PR merges to `main` (or to staging first per release policy).
    - [ ] 3.3.2: CI runs `seed-baseline` against target environment to ensure data state correctness post-migration.
    - [ ] 3.3.3: (Optional) For staging only, execute `seed-fixtures` as needed; never on production.
    - [ ] 3.3.4: Add rollback/runbook: how to disable a migration, revert a release, and re-run seeders safely.
    - [ ] 3.3.5: For environments not using automated DB Branch apply, deploy using `supabase db push` (optionally `--include-seed` for non-production only) after successful CI.

# Constraints & Guardrails
- Language: Go 1.22.x for seeders; SQL for migrations must target Supabase Postgres.
- Dependencies: Use Supabase CLI for schema management; no Flyway usage remains.
- Migrations: DDL-only (schema, RLS, indexes, constraints, functions); do not include data mutations in migrations.
- Seeder Idempotency: Seeders must be safe to run multiple times; use upserts and transactional boundaries.
- Security: No secrets in repo; use CI secrets for `SUPABASE_ACCESS_TOKEN`, DB URLs, and service keys.
- Review: Migration files require code review; existing applied migrations must never be edited—create new ones instead.
- Safety: `seed-fixtures` must never run on production; guard by environment checks and explicit flags.
- Style: Format Go code with `gofmt`; keep SQL readable with consistent casing and statement ordering.
- State: Commit only hand-authored SQL in `supabase/migrations/`; do not commit generated databases or keys.
 - CLI Workflow: Use `supabase migration new`, `supabase migration up`, `supabase db diff`, `supabase db reset`, and `supabase db push` per the official docs.

# Definition of Done (DoD)
- [ ] `supabase/migrations/` contains converted, timestamped migrations equivalent to `db/migrations` (Flyway removed).
- [ ] Supabase DB Branches are enabled; opening a PR creates an isolated branch DB and applies migrations.
- [ ] CI runs `seed-baseline` against PR branch DB and staging; repeat runs are idempotent and verified.
- [ ] (Optional) CI can run `seed-fixtures` for integration tests; never on production.
- [ ] Merging to `main` applies migrations to production; CI subsequently ensures baseline data state.
- [ ] Developer docs updated: local `db diff` flow, how to add migrations, and how to run seeders.
- [ ] All pipeline stages pass green end-to-end.

# References
- Supabase documentation: [Database migrations](https://supabase.com/docs/guides/deployment/database-migrations)

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
- Add comment under header:  `# BLOCKER: <reason>`
- Wait for human intervention.

## Proposing Plan Deviations
- Stop work and set Status → REVIEW_REQUIRED.
- Add a PROPOSAL block with explanation and justification.
- Await human approval; if approved, status resets to READY.


