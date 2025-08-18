---
AIP-ID: 002
Title: Frontend Monorepo Architecture for Arch Blog
Status: IN_PROGRESS
Version: 1.0
Last-Updated: 2025-08-18
---

# Goal
Establish a modern, scalable, and developer-efficient frontend monorepo architecture for Arch Blog that covers tooling, directory structure, development workflow, testing strategy, and CI/CD, with end-to-end type safety integration with the backend API.

# Context
- This plan designs the monorepo under `frontend/`, collaborating with the existing backend (`backend/`) and database workflow (`supabase/`).
- The backend OpenAPI spec at `schema/api.yaml` will be the source of truth for generating types and request functions in `packages/api-client`.
- Use `pnpm workspaces + Turborepo` for project/build orchestration; use `Biome` as the single linter/formatter; use `TypeScript` for end-to-end type safety; and use Storybook to drive UI development.
- Architecture philosophy:
  - Separation of concerns: `apps` are deployable applications; `packages` are reusable libraries.
  - Consistency & standardization: shared TS configs, unified lint/format, unified scripts.
  - Type safety: API schema → generated client → hooks → components → pages, preserving types end-to-end.
  - Comprehensive testing: Unit / Component / E2E layered strategy integrated with CI/CD and caching.

# Phased Plan
- [x] Phase 1: Laying the Foundation
  - [x] 1.1: Initialize the monorepo
    - [x] 1.1.1: In `frontend/`, initialize `package.json` (`pnpm init -y`).
    - [x] 1.1.2: Create `pnpm-workspace.yaml` including `apps/*` and `packages/*`.
    - [x] 1.1.3: Create `.gitignore` (ignore `node_modules`, `.turbo`, `.next`, `dist`, `coverage`).
  - [x] 1.2: Introduce code quality tool (Biome)
    - [x] 1.2.1: Install `@biomejs/biome` as a root devDependency.
    - [x] 1.2.2: Create root `biome.json` with lint/format rules (JS/TS/JSON/Markdown).
    - [x] 1.2.3: Add root scripts: `check`, `lint`, `format` in `package.json`.
  - [x] 1.3: Establish shared TypeScript configuration
    - [x] 1.3.1: Create `packages/tsconfig` package (`package.json` name: `@arch/tsconfig`).
    - [x] 1.3.2: Add `base.json` (`strict: true`, `composite: true`, `skipLibCheck: true`, `moduleResolution: bundler`, etc.).
    - [x] 1.3.3: Provide `react-library.json`, `next-app.json`, `vite-app.json` presets.
  - [x] 1.4: Integrate Turborepo
    - [x] 1.4.1: Install `turbo` (root).
    - [x] 1.4.2: Add `turbo.json` defining `build`, `dev`, `lint`, `test`, `check` pipelines and caching.
    - [x] 1.4.3: Add root scripts that proxy to `turbo run <task>`.
  - [x] 1.5: Scaffold directories and basic package manifests
    - [x] 1.5.1: Create directories: `apps/web-blog`, `apps/admin-panel`, `apps/storybook`, `apps/e2e`; and `packages/ui`, `packages/hooks`, `packages/utils`, `packages/api-client`, `packages/tsconfig`.
    - [x] 1.5.2: For each `apps/*` and `packages/*`, create a minimal `package.json` (`name`, `version`, `private`, `type: module`).
    - [x] 1.5.3: Set root engines for Node 20 LTS and a pnpm version range.
  - [x] 1.6: Baseline acceptance (empty builds allowed)
    - [x] 1.6.1: `pnpm install` succeeds.
    - [x] 1.6.2: `pnpm check` (Biome) succeeds.
    - [x] 1.6.3: `pnpm turbo build` succeeds (even with empty builds).
  - [x] 1.7: Justfile updates (initial, in repo root `justfile`)
    - [x] 1.7.1: Add `frontend:install` → `cd frontend && pnpm install`.
    - [x] 1.7.2: Add `frontend:check` → `cd frontend && pnpm check`.
    - [x] 1.7.3: Add `frontend:lint` → `cd frontend && pnpm turbo run lint`.
    - [x] 1.7.4: Add `frontend:format` → `cd frontend && pnpm turbo run format`.
    - [x] 1.7.5: Add `frontend:build` → `cd frontend && pnpm turbo run build`.
    - [x] 1.7.6: Add `frontend:dev` → `cd frontend && pnpm turbo run dev --parallel`.
    - [x] 1.7.7: Add `frontend:turbo task=<task>` → `cd frontend && pnpm turbo run {{task}}`.
    - [x] 1.7.8: Add `frontend:clean` → remove `frontend/**/node_modules`, `frontend/.turbo`, `frontend/apps/**/.next`, `frontend/apps/**/dist`.
  - [x] 1.8: GitHub Actions updates (frontend workflow skeleton)
    - [x] 1.8.1: Create `.github/workflows/frontend-ci.yml`.
    - [x] 1.8.2: Triggers: `push`/`pull_request` on `main` with path filters: `frontend/**`, `schema/api.yaml`, `.github/workflows/frontend-ci.yml`.
    - [x] 1.8.3: Concurrency: `${{ github.workflow }}-${{ github.ref }}` with `cancel-in-progress: true`.
    - [x] 1.8.4: Add job `check`: Node 20 + `pnpm@9`, install in `frontend/`, run `pnpm check` (Biome).

- [ ] Phase 2: Component-driven development and core UI
  - [ ] 2.1: Create the UI package (`packages/ui`)
    - [ ] 2.1.1: Install React, ReactDOM, TypeScript, TailwindCSS, PostCSS, Autoprefixer.
    - [ ] 2.1.2: Configure `tailwind.config.ts` and `postcss.config.js`; import design tokens (colors/spacing/typography).
    - [ ] 2.1.3: Implement base components: `Button`, `Input`, `Card`; export via `index.ts`.
    - [ ] 2.1.4: Set `tsconfig` to extend `@arch/tsconfig/react-library.json`.
  - [ ] 2.2: Configure Storybook (`apps/storybook`)
    - [ ] 2.2.1: Initialize Storybook (React + Vite).
    - [ ] 2.2.2: Configure `main.ts` to load `packages/ui/**/*.stories.@(ts|tsx)`; enable addons (controls, actions, interactions, a11y).
    - [ ] 2.2.3: Integrate Tailwind by importing global styles in Storybook preview.
    - [ ] 2.2.4: Author stories for `Button`, `Input`, `Card`, covering key props and interactions.
  - [ ] 2.3: Initialize the main app (`apps/web-blog`)
    - [ ] 2.3.1: Bootstrap with `create-next-app` (App Router, TS, ESLint off, using Biome instead).
    - [ ] 2.3.2: Integrate Tailwind; configure `content` to include both `packages/ui` and `apps/web-blog`.
    - [ ] 2.3.3: Import `Button` from `packages/ui` and render it on the home page to validate workspace linking.
  - [ ] 2.4: Acceptance
    - [ ] 2.4.1: Storybook starts and all `packages/ui` components are interactive.
    - [ ] 2.4.2: `apps/web-blog` renders shared components; Tailwind styles apply correctly.
  - [ ] 2.5: Justfile updates (storybook and UI)
    - [ ] 2.5.1: Add `frontend:storybook` → `cd frontend && pnpm --filter @apps/storybook dev`.
    - [ ] 2.5.2: Add `frontend:storybook:build` → `cd frontend && pnpm --filter @apps/storybook build` (or `storybook build`).
    - [ ] 2.5.3: Add `frontend:ui:build` → `cd frontend && pnpm --filter @arch/ui build`.
  - [ ] 2.6: GitHub Actions updates (storybook and build)
    - [ ] 2.6.1: Add job `build`: Node 20 + `pnpm@9`, cache `frontend/.turbo`, run `pnpm turbo run build`.
    - [ ] 2.6.2: Add job `storybook`: run `pnpm --filter @apps/storybook build`; optionally publish to Chromatic if `CHROMATIC_PROJECT_TOKEN` is present.

- [ ] Phase 3: Data layer end-to-end
  - [ ] 3.1: Create the API client (`packages/api-client`)
    - [ ] 3.1.1: Use an OpenAPI generator (`openapi-typescript-codegen` or official `@openapitools`) against `schema/api.yaml` to generate TS types and API functions.
    - [ ] 3.1.2: Provide a configurable `ApiClient` (base URL, interceptors, auth headers).
    - [ ] 3.1.3: Configure build output (ESM + d.ts) and mark as tree-shakeable.
  - [ ] 3.2: Integrate server-state management (TanStack Query)
    - [ ] 3.2.1: In `apps/web-blog`, install `@tanstack/react-query` and set up a `QueryClientProvider` at the app root.
    - [ ] 3.2.2: Rule: all server data flows through React Query; never store it in client-state tools.
  - [ ] 3.3: Create shared data hooks (`packages/hooks`)
    - [ ] 3.3.1: Implement `usePostsQuery` wrapping the API client and React Query (with a clear `queryKey` strategy).
    - [ ] 3.3.2: Export type-safe results and error handling; avoid `any`.
    - [ ] 3.3.3: `packages/hooks` depends on `packages/api-client` and uses the shared TS config.
  - [ ] 3.4: Render data (`apps/web-blog`)
    - [ ] 3.4.1: Use `usePostsQuery` in a page to render a posts list (loading/error/empty/data states).
    - [ ] 3.4.2: Verify the entire type chain from API response to component props.
  - [ ] 3.5: Justfile updates (API client and app dev)
    - [ ] 3.5.1: Add `frontend:api:gen` → `cd frontend && pnpm --filter @arch/api-client gen` (wraps OpenAPI generation).
    - [ ] 3.5.2: Add `frontend:api:clean` → `cd frontend && pnpm --filter @arch/api-client clean`.
    - [ ] 3.5.3: Add `frontend:dev:web-blog` → `cd frontend && pnpm --filter @apps/web-blog dev`.
  - [ ] 3.6: GitHub Actions updates (API generation)
    - [ ] 3.6.1: In job `build`, add step to run `pnpm --filter @arch/api-client gen` (guard with `|| echo "not configured"`).
    - [ ] 3.6.2: Ensure workflow `paths` includes `schema/api.yaml` to retrigger when the API schema changes.

- [ ] Phase 4: App expansion and testing foundation
  - [ ] 4.1: Initialize the admin panel (`apps/admin-panel`)
    - [ ] 4.1.1: Use `create-vite` to initialize a React + Vite + TS app (replace ESLint/Prettier with Biome).
    - [ ] 4.1.2: Integrate Tailwind; share the design system; include `packages/ui` in content scanning.
    - [ ] 4.1.3: Reuse `packages/ui`, `packages/hooks`, `packages/api-client` to build base pages quickly.
  - [ ] 4.2: Introduce unit tests (Vitest)
    - [ ] 4.2.1: Set up Vitest in `packages/utils` and `packages/hooks`; write the first unit tests.
    - [ ] 4.2.2: Add a `test` pipeline with caching in Turborepo.
  - [ ] 4.3: Introduce component tests (Storybook Interaction Tests)
    - [ ] 4.3.1: Use `play` functions in `packages/ui` stories for interaction tests.
    - [ ] 4.3.2: Optionally configure `@storybook/test-runner` in CI.
  - [ ] 4.4: Set up basic CI
    - [ ] 4.4.1: Run `pnpm check` and `pnpm turbo test` in CI (unit/component layers).
    - [ ] 4.4.2: Apply Turborepo caching (local) to reduce rebuild times.
  - [ ] 4.5: Justfile updates (tests and admin app)
    - [ ] 4.5.1: Add `frontend:dev:admin` → `cd frontend && pnpm --filter @apps/admin-panel dev`.
    - [ ] 4.5.2: Add `frontend:test` → `cd frontend && pnpm turbo run test`.
    - [ ] 4.5.3: Add `frontend:test:unit` → `cd frontend && pnpm --filter "@arch/{utils,hooks}" test`.
    - [ ] 4.5.4: Add `frontend:test:components` → `cd frontend && pnpm --filter @arch/ui test` (or Storybook test-runner).
  - [ ] 4.6: GitHub Actions updates (tests)
    - [ ] 4.6.1: Add job `unit-tests`: run `pnpm turbo run test` (Vitest/Storybook test-runner) with Node 20 + `pnpm@9`.
    - [ ] 4.6.2: Set `needs: build` for test jobs to reuse cache and artifacts.

- [ ] Phase 5: End-to-end testing and CI/CD hardening
  - [ ] 5.1: Create the E2E test app (`apps/e2e`)
    - [ ] 5.1.1: Initialize Playwright (TS).
    - [ ] 5.1.2: Configure `playwright.config.ts` to auto-start `web-blog` and `admin-panel` dev servers via `webServer`.
  - [ ] 5.2: Author critical flow tests
    - [ ] 5.2.1: Web Blog: "login → publish post" flow.
    - [ ] 5.2.2: Admin Panel: "login → delete post" flow.
  - [ ] 5.3: Harden CI/CD
    - [ ] 5.3.1: Add an E2E step to CI; ensure Playwright dependencies are installed in the runner.
    - [ ] 5.3.2: Enable Turborepo Remote Caching (e.g., Vercel Remote Cache).
  - [ ] 5.4: (Optional) Visual regression testing
    - [ ] 5.4.1: Integrate Chromatic; run Storybook snapshot comparisons during CI.
  - [ ] 5.5: Justfile updates (E2E)
    - [ ] 5.5.1: Add `frontend:e2e` → `cd frontend && pnpm --filter @apps/e2e test`.
    - [ ] 5.5.2: Add `frontend:e2e:headed` → `cd frontend && pnpm --filter @apps/e2e test:headed`.
    - [ ] 5.5.3: Add `frontend:e2e:ci` → `cd frontend && pnpm --filter @apps/e2e test:ci`.
  - [ ] 5.6: GitHub Actions updates (E2E)
    - [ ] 5.6.1: Add job `e2e`: install Playwright browsers via `pnpm exec playwright install --with-deps` (or dlx fallback).
    - [ ] 5.6.2: Run `pnpm --filter @apps/e2e test:ci` with `CI=true`; set `needs: build`.

Expected final structure (illustrative):

```bash
frontend/
├── apps/
│   ├── web-blog/         # Deployable: Next.js client blog
│   ├── admin-panel/      # Deployable: React (Vite) admin console
│   ├── storybook/        # Deployable: UI workshop & docs
│   └── e2e/              # Deployable: Playwright E2E tests
│
├── packages/
│   ├── ui/               # Library: shared React components (Tailwind)
│   ├── hooks/            # Library: shared React hooks (React Query wrappers)
│   ├── utils/            # Library: shared pure utility functions
│   ├── api-client/       # Library: OpenAPI generated client & types
│   └── tsconfig/         # Library: shared TS configs
│
├── biome.json            # Biome: linter + formatter
├── package.json          # workspace root
├── pnpm-workspace.yaml   # pnpm workspace config
└── turbo.json            # Turborepo pipelines
```

# Constraints & Guardrails
- Language & Engines
  - Node.js 20 LTS; pnpm 9; TypeScript 5.x.
  - TypeScript only; all new code uses `strict: true` and `noUncheckedIndexedAccess`.
- Tooling & Style
  - Biome is the single linter/formatter; do not use ESLint/Prettier concurrently.
  - Turborepo is the only build/task orchestration layer; do not maintain overlapping pipelines.
  - Always extend `@arch/tsconfig`; do not relax configs in subprojects.
- Project Structure & Boundaries
  - `apps/*` must not import each other; do not export code from `apps/*` for reuse.
  - `packages/*` expose stable APIs; avoid cyclic dependencies; use workspace aliases (e.g., `@arch/ui`).
  - Strict separation of concerns: UI (`packages/ui`), data (`packages/api-client`, `packages/hooks`), utilities (`packages/utils`).
- State Management
  - Local UI state: `useState`/`useReducer`.
  - Global client state: use a lightweight store like Zustand for UI concerns only (e.g., theme); never store server data there.
  - Server data: managed exclusively by React Query; do not cache it in client-state tools.
- Data Access & Type Safety
  - `packages/api-client` must be generated from OpenAPI; hand-written types that diverge from the schema are forbidden.
  - `packages/hooks` wraps React Query; apps consume data via these hooks only.
- Styling
  - Tailwind is the primary CSS framework; unify design tokens.
  - `apps/*` should extend the root Tailwind config and expand `content` paths.
- Testing & CI/CD
  - Testing pyramid: Unit (Vitest) → Component (Storybook + RTL) → E2E (Playwright).
  - CI runs `pnpm check` and `pnpm turbo test` on each commit; all checks must pass before merge.
  - Remote caching via Turborepo is allowed; do not commit cache artifacts.

# Definition of Done (DoD)
- [x] Phase 1 outputs:
  - [x] Root `pnpm install` succeeds; `pnpm check` passes; `pnpm turbo build` succeeds.
  - [x] Directory scaffold and shared TS configs are in place; Turborepo pipelines run.
  - [x] Frontend CI workflow exists with `check` job, concurrency, and path filters.
- [ ] Phase 2 outputs:
  - [ ] Storybook displays and interacts with `packages/ui` components; Tailwind styles apply.
  - [ ] `apps/web-blog` renders shared UI components successfully.
  - [ ] CI jobs `build` and `storybook` pass; Chromatic optional publishing works when token present.
- [ ] Phase 3 outputs:
  - [ ] `packages/api-client` generates code and types from `schema/api.yaml`.
  - [ ] `apps/web-blog` shows real backend data via `packages/hooks`, with end-to-end type safety.
  - [ ] CI runs API client generation as part of `build` when `schema/api.yaml` changes.
- [ ] Phase 4 outputs:
  - [ ] `apps/admin-panel` starts successfully and reuses shared packages.
  - [ ] Vitest and Storybook Interaction Tests pass locally and in CI (`unit-tests` job green).
- [ ] Phase 5 outputs:
  - [ ] Playwright E2E tests pass reliably in CI (`e2e` job green); remote cache is effective and build times drop.
  - [ ] (Optional) Chromatic visual regression is part of CI.
- [ ] CI/CD: every commit triggers Lint/Unit/Component/E2E; merging to `main` deploys only affected apps.

# Maintenance Protocol

## Execution Loop
1. Read: Parse the AIP (goal, state, next task).
2. Execute: Perform the first unchecked task.
3. Verify: Confirm success criteria.
4. Update: Check the task and update the `Last-Updated` timestamp.
5. Repeat until everything is complete.

## State Management
- Before starting: set `Status` → `IN_PROGRESS`.
- After all DoD items are done: set `Status` → `COMPLETED`.

## Handling Blockers
- If blocked: stop work, set `Status` → `BLOCKED`, and add at the top of the file:

```markdown
# BLOCKER: <reason>
```

- Resume only after human intervention.

## Proposing Plan Deviations
- If a deviation is desired: stop work, set `Status` → `REVIEW_REQUIRED`, add a `PROPOSAL:` block explaining the change and justification; once approved, set status back to `READY`.


