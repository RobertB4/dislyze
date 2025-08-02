# Testing and Validation Guide

This guide helps you test the Phase 1 deployment and validate that everything is working correctly.

## Pre-Deployment Testing

### 1. Local Docker Build Test
Test the Docker build locally before deploying:

```bash
cd /Users/robert/Documents/dislyze

# Build the Docker image locally
docker build -t lugia-backend-test -f lugia-backend/Dockerfile .

# Run the container locally (optional, requires local database)
# docker run -p 8080:8080 lugia-backend-test
```

### 2. Pulumi Validation

```bash
cd infrastructure

# Install dependencies
npm install

# Validate the Pulumi configuration
pulumi stack select staging --create
pulumi preview

# Check for any configuration errors
pulumi config
```

## Deployment Testing

### 1. Trigger Deployment

**Option A: Push to staging branch**
```bash
git checkout staging
git push origin staging
```

**Option B: Manual workflow trigger**
- Go to GitHub Actions tab in your repository
- Select "Deploy Lugia Backend" workflow
- Click "Run workflow" and select "staging"

### 2. Monitor Deployment

Watch the GitHub Actions workflow:
1. Go to your repository on GitHub
2. Click on "Actions" tab
3. Click on the running workflow
4. Monitor each step for errors

Key steps to watch:
- ✅ Docker image build and push
- ✅ Pulumi infrastructure deployment
- ✅ Health check passes

## Post-Deployment Validation

### 1. Basic Connectivity Tests

```bash
# Get the service URL from Pulumi output
cd infrastructure
SERVICE_URL=$(pulumi stack output lugiaServiceUrl)
echo "Service URL: $SERVICE_URL"

# Test health endpoint
curl -f "$SERVICE_URL/health"
# Expected response: "OK"

# Test CORS and basic API structure
curl -i "$SERVICE_URL/api/health"
# Should return 404 (expected, since /api/health doesn't exist)
# But should show CORS headers in response
```

### 2. Database Connectivity Test

The application should start successfully if database connectivity works. Check Cloud Run logs:

```bash
# View recent logs
gcloud run services logs read dislyze-staging-lugia --region=asia-northeast1 --limit=50

# Look for:
# ✅ "main: API listening on :8080" 
# ✅ Migration logs (if any)
# ❌ Database connection errors
```

### 3. Secret Manager Integration Test

Verify secrets are being loaded correctly:

```bash
# Check that secrets exist
gcloud secrets list

# Verify the Cloud Run service has access (check logs for secret-related errors)
gcloud run services logs read dislyze-staging-lugia --region=asia-northeast1 --limit=50 | grep -i secret
```

### 4. End-to-End API Tests

Test actual API endpoints:

```bash
SERVICE_URL=$(cd infrastructure && pulumi stack output lugiaServiceUrl)

# Test auth endpoints (should work)
curl -X POST "$SERVICE_URL/api/auth/signup" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123","name":"Test User"}'

# Expected: Either success or validation error (both indicate API is working)

# Test protected endpoint (should require auth)
curl "$SERVICE_URL/api/me"
# Expected: 401 Unauthorized
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Docker Build Failures
```bash
# Check if the build context is correct
cd lugia-backend
docker build -t test -f Dockerfile ../ --no-cache

# Common issues:
# - Missing files in build context
# - Go module resolution issues
# - jirachi library not found
```

#### 2. Database Connection Issues
```bash
# Check Cloud SQL instance is running
gcloud sql instances describe dislyze-staging-db

# Check database and user exist
gcloud sql databases list --instance=dislyze-staging-db
gcloud sql users list --instance=dislyze-staging-db

# Test connection from Cloud Shell
gcloud sql connect dislyze-staging-db --user=dislyze
```

#### 3. Secret Access Issues
```bash
# Test secret access with service account
gcloud secrets versions access latest --secret="db-password"

# Check IAM permissions
gcloud secrets get-iam-policy db-password
```

#### 4. Pulumi Deployment Issues
```bash
# Check Pulumi state
cd infrastructure
pulumi stack ls
pulumi stack output

# View detailed deployment logs
pulumi up --logtostderr -v=9

# If state is corrupted, refresh
pulumi refresh
```

#### 5. GitHub Actions Issues
- Check repository secrets are set correctly
- Verify Workload Identity Federation is configured
- Check PULUMI_ACCESS_TOKEN is valid

### Useful Debugging Commands

```bash
# Cloud Run service details
gcloud run services describe dislyze-staging-lugia --region=asia-northeast1

# Live tail logs
gcloud run services logs tail dislyze-staging-lugia --region=asia-northeast1

# Check resource usage
gcloud run services list --region=asia-northeast1

# Database instance details
gcloud sql instances describe dislyze-staging-db

# Secret details
gcloud secrets describe db-password

# Artifact Registry images
gcloud artifacts docker images list asia-northeast1-docker.pkg.dev/dislyze-staging2/dislyze
```

## Success Criteria Checklist

### ✅ Infrastructure Deployed
- [ ] Cloud SQL instance is running
- [ ] Cloud Run service is deployed
- [ ] Artifact Registry contains the image
- [ ] All secrets are created

### ✅ Application Health
- [ ] `/health` endpoint returns "OK"
- [ ] Application logs show successful startup
- [ ] No database connection errors in logs
- [ ] Secrets are being loaded correctly

### ✅ API Functionality
- [ ] Auth endpoints respond (signup/login)
- [ ] Protected endpoints require authentication
- [ ] CORS headers are present
- [ ] Error responses are properly formatted

### ✅ CI/CD Pipeline
- [ ] GitHub Actions workflow completes successfully
- [ ] Docker image builds and pushes
- [ ] Pulumi deployment succeeds
- [ ] Automated health checks pass

## Next Steps After Validation

Once all tests pass:

1. **Create staging branch**: Make staging branch the main deployment branch
2. **Configure domain**: Set up `staging-app.flownavi.com` (for Phase 4)
3. **Monitor logs**: Set up log monitoring and alerting
4. **Plan Phase 2**: Frontend embedding implementation

## Performance Baseline

After successful deployment, establish performance baselines:

```bash
# Response time test
time curl -s "$SERVICE_URL/health"

# Load test (optional)
# Use tools like wrk or ab to test basic load handling
```

Expected performance:
- Health endpoint: < 100ms response time
- Cold start: < 5 seconds for first request
- Memory usage: < 128MB steady state
- CPU usage: Minimal at rest

This completes Phase 1 validation. The system should now be ready for Phase 2 enhancements.