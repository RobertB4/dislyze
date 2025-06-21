# CLAUDE.md - Testing Guidelines

This file provides specific guidance for Claude Code when working with tests in lugia-backend.

## Research-First Principle

**ALWAYS research existing code before making assumptions or implementing new functionality.**

1. **Check existing patterns**: Look at similar test files to understand structure and conventions
2. **Verify helper functions**: Search for existing helper functions before creating new ones
3. **Confirm endpoints**: Always verify API endpoints by checking router configuration
4. **Follow established patterns**: Use existing test utilities and patterns rather than creating duplicates

## API Endpoint Discovery

Before writing tests for any endpoint:

1. **Check router configuration**: Look in `main.go` or routing files to find the exact URL paths
2. **Verify base URL**: Check `test/integration/setup/helpers.go` for `BaseURL` constant
3. **Confirm HTTP methods**: Verify GET/POST/PUT/DELETE from router configuration
4. **Example endpoint discovery process**:
   ```bash
   # Search for route definitions
   grep -r "ip-whitelist" main.go
   # Check base URL in test setup
   grep -r "BaseURL" test/integration/setup/
   ```

## Integration Testing Patterns

### Test File Structure

Integration tests follow this structure:
```go
func TestFeatureNameIntegration(t *testing.T) {
    pool := setup.InitDB(t)
    // Reset database state at the start of the test function if needed
    setup.ResetAndSeedDB(t, pool)
    defer setup.CloseDB(pool)
    
    client := &http.Client{}
    
    t.Run("test_case_name", func(t *testing.T) {
        // Reset database state before each test if needed
        // Only reset once. either once per function or once per test case depending on the test requirements
        setup.ResetAndSeedDB(t, pool)
        
        // Test implementation
    })
}
```

### Test Setup Patterns

1. **Database Reset**: Always call `setup.ResetAndSeedDB(t, pool)` at the start of each test
2. **User Authentication**: Use `setup.LoginUserAndGetTokens(t, email, password)` for auth
3. **Permission Setup**: Create roles and assign permissions using existing helpers
4. **Enterprise Features**: Configure tenant features using helper functions

### URL Construction

**NEVER hardcode or guess URLs**. Always:
1. Check `setup.BaseURL` in `test/integration/setup/helpers.go`
2. Verify the endpoint path from router configuration
3. Construct URLs using: `fmt.Sprintf("%s/endpoint-path", setup.BaseURL)`

**Example**: If `BaseURL = "http://lugia-backend:13001/api"` and endpoint is `/ip-whitelist/create`:
```go
reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
// Results in: "http://lugia-backend:13001/api/ip-whitelist/create"
```

### Error Response Expectations

Some endpoints return **status codes only** (no error message body):
- Authentication/authorization failures: 401/403 with no body
- Validation failures: 400 with no body  
- Success responses: 200 with no body (for mutations)

**How to identify**: Check the endpoint implementation in `features/` directory to see if it uses:
- `responder.RespondWithError(w, appErr)` - includes error message
- `w.WriteHeader(http.StatusXXX)` - status only

### Test Data Management

1. **Use seed data constants**: Reference `setup.TestTenantsData["enterprise"].ID` instead of hardcoding UUIDs
2. **Permission IDs**: Find permission IDs in database migration files or existing tests
3. **Test users**: Use predefined users from `setup.TestUsersData` map

### Common Test Patterns

#### Authentication Tests
```go
// No auth token
req, err := http.NewRequest("POST", reqURL, body)
// Expect: 401 Unauthorized

// With valid auth
accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})
```

#### Permission Tests
```go
// Create role with specific permissions
roleID := createIPWhitelistRole(t, pool, tenantID, []string{permissionID})
assignRoleToUser(t, pool, userID, roleID, tenantID)
```

#### Feature Flag Tests
```go
// Enable/disable enterprise features
updateTenantEnterpriseFeatures(t, pool, tenantID, map[string]interface{}{
    "feature_name": map[string]interface{}{
        "enabled": true,
        "active": false,
        "other_config": "value",
    },
})
```

## Avoiding Duplicate Code

1. **Check existing test files** in the same package for similar helper functions
2. **Search for function names** before creating new ones:
   ```bash
   grep -r "func functionName" test/integration/package_name/
   ```
3. **Reuse test utilities** from `test/integration/setup/` package
4. **Follow DRY principle** - if you see repeated code patterns, look for existing abstractions

## Test Execution

- **Single test function**: `make test-integration-single TEST=TestFunctionName`
- **All integration tests**: `make test-integration`
- **Debug failures**: Check Docker logs and database state

## Best Practices

1. **Research first, implement second**
2. **Verify endpoints before testing**
3. **Use existing helpers and patterns**
4. **Follow established test structure**
5. **Reset database state between tests**
6. **Use seed data constants, not hardcoded values**
7. **Check error response expectations**
8. **Test both success and failure cases**
9. **Ensure tenant isolation in multi-tenant tests**
10. **Write descriptive test names that explain the scenario**