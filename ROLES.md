✅ Phase 1: Settings Tab Integration & Core Structure (COMPLETED)

  Files created/modified:
  - ✅ lugia-frontend/src/routes/settings/SettingsTabs.svelte -
  Added roles tab with roles.view permission
  - ✅ lugia-frontend/src/routes/settings/roles/+page.ts - Data
  loading from /api/roles
  - ✅ lugia-frontend/src/routes/settings/roles/+page.svelte - Main
  page with role listing
  - ✅ lugia-frontend/src/routes/settings/roles/Skeleton.svelte -
  Realistic loading skeleton

  Completed features:
  - Role listing with proper ordering: 管理者, 編集者, 閲覧者, then
  custom roles by name
  - Complete table layout with permissions badges and tooltips
  - Default/Custom role type indicators
  - Action buttons (placeholder for now)
  - Proper Japanese text and consistent styling

  ---
  🔄 Phase 2: Create Role Functionality (NEXT)

  Features to implement:
  1. Slideover form for role creation with fields:
    - Name (required, trimmed)
    - Description (optional, trimmed)
    - Permission selection using radio button groups
  2. Permission selection UI:
    - Three radio button groups: Users, Roles, Tenant
    - Options per group: None, View, Edit
  3. Validation with felte:
    - Name required
    - At least one permission must be non-"none"
  4. API integration: POST to /api/roles/create with hardcoded
  permission mapping

  ---
  📋 Phase 3: Update Role Functionality

  Features to implement:
  1. Edit buttons for custom roles only (disabled for default roles)
  2. Pre-populated form with existing role data
  3. Same validation as create
  4. Permission checking (roles.edit)
  5. API integration: POST to /api/roles/{roleID}/update

  ---
  🗑️ Phase 4: Delete Role Functionality

  Features to implement:
  1. Delete buttons for custom roles only (disabled for default
  roles)
  2. Confirmation modal requiring exact role name input
  3. Warning about role being assigned to users
  4. API integration: POST to /api/roles/{roleID}/delete
  5. Graceful error handling for roles in use (handled by
  mutationFetch)

  ---
  🧪 Phase 5: E2E Testing

  File to create:
  - lugia-frontend/test/e2e/settings/roles.spec.ts

  Test scenarios:
  1. Authentication and access control
  2. Role creation with validation
  3. Role editing restrictions
  4. Role deletion with confirmation
  5. Permission selection validation
  6. Error handling