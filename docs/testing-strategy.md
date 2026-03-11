# Testing Strategy

## Requirements by Phase

### All Phases
- **Backend API endpoints**: Integration test per endpoint (happy path + key error cases)
- **Backend services/repositories**: Unit tests for business logic
- **Frontend components**: Basic render test for new components
- **Frontend pages**: Smoke test ensuring page renders without crash

### Test Commands
- Backend: `cd backend && go test -v -race ./...`
- Frontend: `cd frontend && pnpm test:run`
- Full check: `pnpm lint && pnpm type-check && pnpm test`

### Coverage Expectations
- New backend packages: ≥ 70% line coverage
- New frontend components: At least 1 render test per component
- API endpoints: At least happy-path test per public endpoint

### Test File Conventions
- Backend: `*_test.go` in same package, or `package_test` for integration tests
- Frontend: `*.test.tsx` co-located with component file

### Integration Test Pattern (Backend)
Use in-memory SQLite for handler/service tests:
```go
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    db.AutoMigrate(&model.Article{}, ...)
    return db
}
```

### Frontend Test Pattern
```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

describe("ComponentName", () => {
  it("renders without crash", () => {
    render(<ComponentName />);
    expect(screen.getByText("expected text")).toBeInTheDocument();
  });
});
```
