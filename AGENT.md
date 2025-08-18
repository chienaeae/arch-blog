# Arch Blog - Codebase Guide

## For Agents

- Read the aip/aip-guide.md, aip/aip-template.md files to understand how to read, execute, and maintain the AIP.

## Project Overview

Full-stack blog platform with:
- **Backend**: Production-ready Go API using Hexagonal Architecture with event-driven communication between bounded contexts
- **Frontend**: Modern monorepo architecture using pnpm workspaces, Turborepo, and TypeScript for end-to-end type safety

## Architecture & Patterns

### Backend Architecture
- **Hexagonal (Ports & Adapters)**: Domain logic isolated from infrastructure
- **Bounded Contexts**: Users, Authorization, Posts, Themes - communicate via event bus
- **Consumer-Driven Contracts**: Cross-context adapters without direct dependencies
- **Contract-First API**: OpenAPI spec → oapi-codegen (chi-server) → implementation

### Backend Patterns
- **Repository Pattern**: Pure data mappers in `ports/`, implementations in `adapters/postgres/`
- **Service Layer**: Business logic in `application/`, orchestrates repositories & events
- **Domain Models**: Self-validating entities in `domain/` with business rules
- **Platform Services**: Cross-cutting concerns in `platform/` (errors, events, config)
- **Middleware Chains**: JWT → AuthAdapter → Authorization → Ownership checks

### Frontend Architecture
- **Monorepo**: pnpm workspaces + Turborepo for build orchestration
- **Apps**: Deployable applications (web-blog, admin-panel, storybook, e2e)
- **Packages**: Reusable libraries (ui, hooks, utils, api-client, tsconfig)
- **Type Safety**: OpenAPI → TypeScript client → React hooks → Components
- **Code Quality**: Biome for unified linting and formatting

## Project Structure

### Backend
```
backend/
├── cmd/api/              # Entry point
├── internal/
│   ├── adapters/         # Infrastructure implementations
│   │   ├── api/          # OpenAPI generated code
│   │   ├── auth/         # JWT middleware
│   │   ├── authz_adapter/# Cross-context authorization
│   │   ├── postgres/     # Repository implementations
│   │   └── rest/         # HTTP handlers
│   ├── platform/         # Shared infrastructure
│   │   ├── apperror/     # Structured error system
│   │   ├── eventbus/     # In-process pub/sub & request/reply
│   │   ├── ownership/    # Resource ownership registry
│   │   └── validator/    # Validation utilities
│   ├── [context]/        # Bounded contexts (users, authz, posts, themes)
│   │   ├── application/  # Use cases/services
│   │   ├── domain/       # Entities & business logic
│   │   └── ports/        # Interface contracts
│   └── server/           # DI setup (Wire)
```

### Frontend
```
frontend/
├── apps/
│   ├── web-blog/         # Next.js client blog (planned)
│   ├── admin-panel/      # React (Vite) admin console (planned)
│   ├── storybook/        # UI component workshop (planned)
│   └── e2e/              # Playwright E2E tests (planned)
├── packages/
│   ├── ui/               # Shared React components (planned)
│   ├── hooks/            # Shared React hooks (planned)
│   ├── utils/            # Shared utility functions (planned)
│   ├── api-client/       # OpenAPI generated TypeScript client (planned)
│   └── tsconfig/         # Shared TypeScript configurations
├── biome.json            # Linter & formatter configuration
├── turbo.json            # Turborepo pipeline configuration
└── pnpm-workspace.yaml   # pnpm workspace configuration
```

## Key Components & Tools

### Frontend Tools
- **pnpm**: Fast, disk space efficient package manager with workspaces
- **Turborepo**: Build system for JavaScript/TypeScript monorepos with caching
- **Biome**: Fast formatter and linter for JavaScript, TypeScript, and JSON
- **TypeScript**: Strict mode enabled with `noUncheckedIndexedAccess` for maximum type safety

### Backend Platform Services
- **AppError System** (`platform/apperror/`): Dual-level error codes (system + business)
- **Event Bus** (`platform/eventbus/`): Pub/sub and request/reply between contexts
- **Ownership Registry** (`platform/ownership/`): Resource-based permission checks
- **Transaction Utils** (`platform/postgres/`): Service-layer transaction management

### Code Generation
- **OpenAPI → Go**: `just gen` - generates chi-server stubs from `schema/api.yaml`
- **Wire DI**: `just wire` - generates dependency injection code
- **Generated files**: `adapters/api/generated.go`, `server/wire_gen.go` (gitignored)
- **Important**: Run `just generate` after pulling changes to ensure generated files are up to date before running tests.

### Database
- **Driver**: `pgx/v5` (Direct PostgreSQL driver).
- **Schema Management**: Handled by **Supabase CLI**.
  - Migrations are located in `supabase/migrations/`.
  - Migrations use a `YYYYMMDDHHMMSS_description.sql` timestamp format.
- **Local Development**:
  - `supabase/seed.sql` is used to seed the local database for development. This is run automatically by `just db-reset`.
  - For initial setup and linking to the Supabase platform, see `supabase/README.md`.
- **Best Practices**:
  - Use `pgtype.UUID`, `pgtype.Text`, `pgtype.Timestamptz`.
  - Defer rollback: `defer func() { _ = tx.Rollback(ctx) }()`.
  - Transactions at service layer, not repository.

### HTTP & Middleware
- **chi v5**: Router with per-route middleware
- **Middleware Chain**: JWT → AuthAdapter → Authorization → Handler
- **BaseHandler**: Common utilities (error handling, JSON responses, UUID parsing)
- **Context Keys**: `UserIDKey`, `UserEmailKey` for internal identity

## Development Workflow

### Commands (justfile)
```bash
# Backend
just check        # Format, test, and lint all backend code
just run          # Start API server
just test         # Run tests with race detector
just lint         # Run golangci-lint

# Code Generation
just generate     # Generate all code (OpenAPI + Wire)
just gen          # Generate from OpenAPI
just wire         # Generate DI code

# Database (Supabase)
just db-migrate   # Apply all migrations to the local DB
just db-reset     # Reset the local DB and re-apply all migrations + seed.sql
just db-diff ...  # Diff the local DB against migrations (e.g., `just db-diff -f my-change`)

# Frontend
just frontend-install  # Install frontend dependencies
just frontend-check    # Run Biome checks
just frontend-lint     # Run linting across all packages
just frontend-format   # Format all frontend code
just frontend-build    # Build all frontend packages
just frontend-dev      # Start dev servers in parallel
just frontend-clean    # Clean frontend build artifacts
```

### Environment Variables
- `DATABASE_URL`: PostgreSQL connection
- `JWKS_ENDPOINT`: JWT validation endpoint
- `JWT_ISSUER`: Expected JWT issuer
- `PORT`: Server port (default: 8080)

## Authorization Model
- **Permissions**: `resource:action:scope` (e.g., `posts:update:own`)
- **Scopes**: `own` (ownership), `any` (global), or empty
- **Roles**: System, template, and custom roles
- **Ownership**: Each resource registers ownership checker

## API Structure
- **Health**: `/api/v1/health/{live,ready}`
- **Users**: Profile management with JWT auth
- **Authorization**: Role & permission management
- **Posts**: CRUD with status transitions (draft/published/archived)
- **Themes**: Article collections with curator ownership

## Testing Strategy

### Backend Testing
- **Domain Tests**: Business logic validation
- **Repository Tests**: Database integration
- **Service Tests**: Use case orchestration
- **Handler Tests**: HTTP layer translation

### Frontend Testing (Planned)
- **Unit Tests**: Vitest for utility functions and hooks
- **Component Tests**: Storybook interaction tests
- **E2E Tests**: Playwright for critical user flows

## Code Quality Rules

### Backend
- No business logic in repositories
- Context keys as typed constants
- Always check errors, even `Close()`
- Prefer standard library over custom code
- Transaction management at service layer
- Domain models self-validate

### Frontend
- TypeScript `strict` mode always enabled
- No `any` types without explicit justification
- Shared components in `packages/ui` only
- Server state managed exclusively by React Query
- All code formatted with Biome

## Git Commit Convention
No AI-generated wording in commit messages (configured in global CLAUDE.md)