# Contributing to Exchange Simulator

## Development Environment

### Prerequisites
- **Go**: 1.24+
- **Protocol Buffers**: protoc compiler with Go plugin
- **Docker**: For integration testing

### Setup
```bash
# Install dependencies
go mod download

# Generate protobuf code (if applicable)
make proto-gen

# Run tests
go test ./...
```

## Workflow

### 1. Branch Creation
**Branch Naming Convention**: `type/epic-XXX-9999-milestone-description`

Examples:
- `feature/epic-TSE-0001-foundation-add-grpc-health-check`
- `fix/epic-TSE-0002-trading-fix-order-validation`
- `chore/epic-TSE-0001-foundation-update-dependencies`

### 2. Development
**Before committing**:
1. Run tests: `go test ./...`
2. Run linting: `golangci-lint run`
3. Verify builds: `go build ./...`
4. Update TODO.md if working on milestone tasks

### 3. Commit Messages
Follow conventional commits with epic tracking:

```
type(epic-XXX/milestone): description

Detailed explanation if needed

Milestone: Milestone Name
Behavior: What this enables
```

## Code Standards

### Clean Architecture Rules
1. **Domain Layer**: NO external dependencies
2. **Application Layer**: Depends on domain only
3. **Infrastructure Layer**: Implements ports from domain

### Testing Requirements
- Minimum 30% test coverage
- Unit tests for domain logic
- Integration tests for external dependencies
- Table-driven tests preferred

### Go-Specific Standards
- Follow effective Go guidelines
- Use context for cancellation
- Handle errors explicitly
- Document exported types and functions

## Pull Requests

Before creating PR:
1. ✅ All tests passing
2. ✅ Linting clean
3. ✅ PR documentation created in `docs/prs/`
4. ✅ TODO.md updated
5. ✅ No markdown linting errors

See git_workflow_checklist skill for full PR requirements.

## Questions?

Check project documentation:
- Architecture: `.claude/.claude_architecture.md`
- Go Standards: `.claude/.claude_go.md`
- Testing: `.claude/.claude_testing_go.md`
