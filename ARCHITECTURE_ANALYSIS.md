# Backend Architecture Analysis Report

This report provides a comprehensive analysis of the project's backend architecture, a professional opinion on its design, and a step-by-step guide for implementing a new feature (an audit log) in line with the existing patterns.

## 1. Architecture Overview

The backend is a modern Go application meticulously structured around the **Clean Architecture** (also known as Hexagonal or Ports and Adapters) pattern. This architectural style emphasizes a clear separation of concerns, resulting in a system that is modular, independent of external frameworks, and highly testable.

The key characteristics of this architecture are:

### a. Schema-First API with OpenAPI

The project employs a **schema-first** approach to API development. The entire API contract—endpoints, request/response models, and security schemes—is formally defined in the `schema/api.yaml` file using the OpenAPI 3.0 specification.

- **Benefit**: This creates a single source of truth for the API, enabling clear communication between frontend and backend teams. It also allows for the use of code generation tools (`oapi-codegen`, as seen in `go.mod`) to automatically create server stubs, request validation logic, and data models, which significantly reduces boilerplate code and ensures the implementation stays synchronized with the documentation.

### b. Compile-Time Dependency Injection with Google Wire

Dependency Injection (DI) is managed using `github.com/google/wire`. Unlike runtime DI frameworks, Wire is a compile-time tool. The file `backend/internal/server/wire.go` explicitly defines how all the application's components are constructed and wired together.

- **Benefit**: This approach provides strong compile-time safety. Any errors in the dependency graph (e.g., a missing dependency or a type mismatch) are caught during compilation, not at runtime. It also makes the application's startup logic and component relationships explicit and easy to trace.

### c. Clear Architectural Layers

The codebase is organized into distinct, well-defined layers, enforcing the dependency rule (dependencies flow inwards):

- **Domain Layer (`internal/.../domain`)**: The core of the application. It contains the fundamental business entities and their intrinsic rules (e.g., `User`, `Role`). This layer is completely independent and has no knowledge of the layers outside it.

- **Application Layer (`internal/.../application`)**: This layer orchestrates the use cases of the application. It contains the core business logic (e.g., `AuthzService`, `UserService`). It depends on interfaces (ports) defined for repositories or other external services but knows nothing about their concrete implementations (like PostgreSQL or a specific payment gateway).

- **Adapter Layer (`internal/adapters`)**: This layer contains the concrete implementations of the ports defined by the application layer. It acts as a bridge between the application core and external technologies. Key adapters in this project include:
    - `postgres`: Implements the repository interfaces for data persistence using a PostgreSQL database.
    - `rest`: Contains the HTTP handlers and middleware, adapting incoming HTTP requests to calls on the application services.
    - `auth`: Provides JWT authentication middleware.

- **Frameworks & Drivers Layer (`cmd/api`, `internal/server`)**: This is the outermost layer, responsible for initializing and starting the application. It sets up the server, database connections, configuration (`viper`), and logging, and wires everything together using the definitions in `wire.go`.

### d. Modular Structure

The business logic is further broken down into modules or "bounded contexts," such as `users` and `authz` (authorization). Each module is self-contained with its own domain, application, and port definitions. This modularity makes the system easier to navigate and allows new features to be developed in isolation without affecting other parts of the application.

## 2. Professional Opinion on the Architecture

This backend architecture is of a very high standard and demonstrates a mature understanding of modern software engineering principles.

### Strengths

- **Maintainability & Scalability**: The strict separation of concerns and modular design make the codebase exceptionally easy to maintain and scale. Adding new features or changing existing ones can be done with high confidence and minimal risk of unintended side effects.
- **Testability**: The architecture is highly conducive to testing. The domain and application layers can be unit-tested in complete isolation from the database or web server by using mock implementations of the repository interfaces. This leads to fast, reliable tests.
- **Flexibility**: The "Ports and Adapters" pattern makes the application resilient to technological change. For example, if the team decided to switch from PostgreSQL to another database, only the `postgres` adapter would need to be replaced with a new adapter; the core application and domain logic would remain untouched.
- **Clarity and Explicitness**: The combination of a schema-first API and compile-time DI makes the system's structure and behavior remarkably clear. There is little "magic" involved, which reduces the learning curve for new developers joining the project.

### Potential Considerations

- **Boilerplate**: The primary trade-off of this architecture is the amount of initial setup and boilerplate required. For very small, simple projects or short-lived microservices, it might be considered over-engineering. However, for a complex, long-term application, this initial investment pays significant dividends in maintainability.
- **Discipline**: The architecture's success relies on the development team's discipline to adhere to its rules, particularly the dependency rule. A single import statement from an inner layer to an outer layer can begin to erode the benefits.

## 3. Guide: Implementing a General Audit Log Feature

Here is a step-by-step guide to adding a general-purpose audit log feature, designed to fit seamlessly into the existing architecture.

### Step 1: Define the Audit Domain

Create a new domain package for auditing.

- **Create file `backend/internal/audit/domain/audit.go`**:
  ```go
  package domain

  import (
	  "time"
	  "github.com/google/uuid"
  )

  // Action defines the type of action that was performed.
  type Action string

  const (
	  RoleCreated    Action = "role.created"
	  RoleDeleted    Action = "role.deleted"
	  UserRoleAssigned Action = "user.role.assigned"
	  UserRoleRevoked  Action = "user.role.revoked"
	  // ... add other actions as needed
  )

  // AuditLog represents a single entry in the audit trail.
  type AuditLog struct {
	  ID         uuid.UUID
	  ActorID    uuid.UUID // ID of the user who performed the action
	  Action     Action
	  TargetID   uuid.UUID // ID of the entity that was affected (e.g., the user or role)
	  TargetType string    // Type of the entity (e.g., "user", "role")
	  Details    map[string]interface{} // For storing old/new values
	  Timestamp  time.Time
  }
  ```

### Step 2: Define the Port (Repository Interface)

- **Create file `backend/internal/audit/ports/repository.go`**:
  ```go
  package ports

  import (
	  "context"
	  "github.com/philly/arch-blog/backend/internal/audit/domain"
  )

  // AuditRepository defines the persistence interface for audit logs.
  type AuditRepository interface {
	  Create(ctx context.Context, log *domain.AuditLog) error
  }
  ```

### Step 3: Implement the Persistence Adapter (PostgreSQL)

- **Create a database migration file `db/migrations/V3__create_audit_logs_table.sql`**:
  ```sql
  CREATE TABLE audit_logs (
      id UUID PRIMARY KEY,
      actor_id UUID NOT NULL,
      action VARCHAR(255) NOT NULL,
      target_id UUID NOT NULL,
      target_type VARCHAR(255) NOT NULL,
      details JSONB,
      "timestamp" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL
  );

  CREATE INDEX idx_audit_logs_target ON audit_logs(target_type, target_id);
  CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id);
  ```

- **Create file `backend/internal/adapters/postgres/audit_repository.go`**:
  ```go
  package postgres

  import (
	  "context"
	  "github.com/jackc/pgx/v5/pgxpool"
	  "github.com/philly/arch-blog/backend/internal/audit/domain"
	  "github.com/philly/arch-blog/backend/internal/audit/ports"
  )

  type AuditRepository struct {
	  db *pgxpool.Pool
  }

  func NewAuditRepository(db *pgxpool.Pool) ports.AuditRepository {
	  return &AuditRepository{db: db}
  }

  func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	  query := `
		  INSERT INTO audit_logs (id, actor_id, action, target_id, target_type, details, timestamp)
		  VALUES ($1, $2, $3, $4, $5, $6, $7)
	  `
	  _, err := r.db.Exec(ctx, query,
		  log.ID, log.ActorID, log.Action, log.TargetID, log.TargetType, log.Details, log.Timestamp,
	  )
	  return err
  }
  ```

### Step 4: Create the Application Service

This service will provide a simple interface for other parts of the application to create audit logs.

- **Create file `backend/internal/audit/application/service.go`**:
  ```go
  package application

  import (
	  "context"
	  "time"
	  "github.com/google/uuid"
	  "github.com/philly/arch-blog/backend/internal/audit/domain"
	  "github.com/philly/arch-blog/backend/internal/audit/ports"
  )

  type AuditService struct {
	  repo ports.AuditRepository
  }

  func NewAuditService(repo ports.AuditRepository) *AuditService {
	  return &AuditService{repo: repo}
  }

  func (s *AuditService) Log(ctx context.Context, actorID, targetID uuid.UUID, targetType string, action domain.Action, details map[string]interface{}) error {
	  logEntry := &domain.AuditLog{
		  ID:         uuid.New(),
		  ActorID:    actorID,
		  Action:     action,
		  TargetID:   targetID,
		  TargetType: targetType,
		  Details:    details,
		  Timestamp:  time.Now(),
	  }
	  return s.repo.Create(ctx, logEntry)
  }
  ```

### Step 5: Wire Everything Together

Now, integrate the new audit module into the dependency injection graph.

- **Create `backend/internal/audit/provider.go`**:
  ```go
  package audit

  import (
	  "github.com/google/wire"
	  "github.com/philly/arch-blog/backend/internal/audit/application"
  )

  // Using the existing postgres provider set for the repository
  var ProviderSet = wire.NewSet(
	  application.NewAuditService,
  )
  ```

- **Update `backend/internal/adapters/postgres/provider.go`**:
  Add the new `NewAuditRepository` and bind its interface.
  ```go
  // ... existing code
  import (
      // ...
      "github.com/philly/arch-blog/backend/internal/audit/ports"
  )

  var ProviderSet = wire.NewSet(
      NewUserRepository,
      wire.Bind(new(userPorts.UserRepository), new(*UserRepository)),
      NewRoleRepository,
      wire.Bind(new(authzPorts.RoleRepository), new(*RoleRepository)),
      NewPermissionRepository,
      wire.Bind(new(authzPorts.PermissionRepository), new(*PermissionRepository)),

      // Add the following lines
      NewAuditRepository,
      wire.Bind(new(ports.AuditRepository), new(*AuditRepository)),
  )
  ```

- **Update `backend/internal/server/wire.go`**:
  Add the new `audit.ProviderSet` to the `wire.Build` call.
  ```go
  // ... imports
  import (
      // ...
      "github.com/philly/arch-blog/backend/internal/audit"
  )
  // ...
  func InitializeApp(ctx context.Context) (*App, func(), error) {
      wire.Build(
          // ... existing providers

          // Application services
          application.ProviderSet,
          authzApp.ProviderSet,
          audit.ProviderSet, // <-- Add this line

          // ... rest of the providers
      )
      return nil, nil, nil
  }
  ```

### Step 6: Integrate with Business Logic

Finally, inject the `AuditService` into the application services where you want to log actions.

- **Example: Modify `backend/internal/authz/application/service.go`**:
  ```go
  package application

  import (
      // ... other imports
      "github.com/philly/arch-blog/backend/internal/audit/application"
      "github.com/philly/arch-blog/backend/internal/audit/domain"
      "github.com/philly/arch-blog/backend/internal/platform/auth"
  )

  type AuthzService struct {
      roleRepo       ports.RoleRepository
      permissionRepo ports.PermissionRepository
      auditService   *application.AuditService // <-- Add dependency
  }

  // Update NewAuthzService to accept the new dependency
  func NewAuthzService(roleRepo ports.RoleRepository, permissionRepo ports.PermissionRepository, auditService *application.AuditService) *AuthzService {
      return &AuthzService{
          roleRepo:       roleRepo,
          permissionRepo: permissionRepo,
          auditService:   auditService, // <-- Store dependency
      }
  }

  // Update a method to log an audit event
  func (s *AuthzService) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) (*authzDomain.UserRole, error) {
      // ... existing logic to assign the role ...

      // Get actor from context
      actor, err := auth.UserFromContext(ctx)
      if err != nil {
          return nil, err // Or handle error appropriately
      }

      // Log the audit event on success
      details := map[string]interface{}{"assigned_role_id": roleID}
      err = s.auditService.Log(ctx, actor.ID, userID, "user", domain.UserRoleAssigned, details)
      if err != nil {
          // Decide how to handle logging failure. Usually, you don't want to
          // fail the main operation. Just log the error.
          // s.log.Error("failed to create audit log", "error", err)
      }

      return userRole, nil
  }
  ```
- **Update `backend/internal/authz/provider.go`**: The signature of `NewAuthzService` has changed, so `wire` will now automatically inject the `AuditService` dependency. No changes are needed here, as `wire` resolves the new dependency.

After these steps, you would need to run `go generate ./...` or `wire ./...` in the `backend/internal/server` directory to regenerate the `wire_gen.go` file, which will contain the updated dependency graph.
