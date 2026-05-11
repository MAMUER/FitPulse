# ADR 0010: Database Schema and Service Layer Refactoring

## Context

The biometric and training services required improvements to data consistency, test coverage, and service reliability. The existing repository patterns lacked proper timestamp tracking, and the test coverage was insufficient for critical path validation.

## Decision

1. **Biometric Repository Enhancement**
   - Added `created_at` timestamp field to the Save method in `biometric_repository.go`
   - Ensures all stored biometric records have creation timestamps for audit and debugging

2. **Training Service Schema Expansion**
   - Extended training service data models with additional fields and relationships
   - Improved data representation for training plans and progress tracking

3. **Test Coverage Improvement**
   - Added comprehensive unit tests for data processor with environment variable handling
   - Created integration tests for training service (GeneratePlan, GetProgress)
   - Implemented mock database interactions for isolated unit testing

4. **Dependency Cleanup**
   - Removed unused `postgres` module dependency from `go.mod`

## Consequences

- **Positive**: Better data consistency with automatic timestamp tracking
- **Positive**: Comprehensive test coverage (unit + integration) improves reliability
- **Positive**: Mock-based unit tests enable faster development cycles
- **Positive**: Cleaner dependency graph reduces build times and security surface
- **Neutral**: Database schema migration required for existing data

## Implementation

- Modified `internal/repository/biometric_repository.go`
- Updated `cmd/biometric-service/biometric_service_test.go`
- Added `cmd/data-processor/data_processor_unit_test.go`
- Created `cmd/training-service/training_service_integration_test.go`
- Added `cmd/training-service/training_service_unit_test.go`
- Updated `go.mod` to remove unused dependency

## Alternatives Considered

- Using database triggers for timestamps: Adds database coupling, less portable
- Relying solely on integration tests: Slower feedback, harder to debug
- Keeping unused dependencies: Increases binary size and vulnerability surface
