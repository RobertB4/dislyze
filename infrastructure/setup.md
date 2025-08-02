# Infrastructure Setup Guide

This guide will help you set up the Google Cloud infrastructure and secrets for the Dislyze deployment.

## Prerequisites

- Google Cloud SDK (`gcloud`) installed and configured
- Pulumi CLI installed
- Node.js 20+ installed
- Access to the `dislyze-staging2` Google Cloud project

## Step 1: Set up Google Cloud Configuration

```bash
# Set the project
gcloud config set project dislyze-staging2

# Ensure you're authenticated
gcloud auth login

# Enable required APIs (if not already enabled)
gcloud services enable \
  run.googleapis.com \
  sql-component.googleapis.com \
  sqladmin.googleapis.com \
  compute.googleapis.com \
  secretmanager.googleapis.com \ 
  artifactregistry.googleapis.com \
  iam.googleapis.com \
  cloudresourcemanager.googleapis.com
```

## Step 2: Create Secret Manager Secrets

You'll need to create the following secrets with your actual values:

```bash
# Database password (generate a strong password)
echo -n "" | gcloud secrets create db-password --data-file=-

# JWT secrets
echo -n "" | gcloud secrets create auth-jwt-secret --data-file=-
echo -n "" | gcloud secrets create create-tenant-jwt-secret --data-file=-
echo -n "" | gcloud secrets create ip-whitelist-emergency-jwt-secret --data-file=-

# Passwords
echo -n "" | gcloud secrets create initial-pw --data-file=-
echo -n "" | gcloud secrets create internal-user-pw --data-file=-

# SendGrid API key
echo -n "" | gcloud secrets create sendgrid-api-key --data-file=-
```

**Important**: Replace the placeholder values with your actual secrets!

## Step 3: Set up Workload Identity Federation

Create a Workload Identity Pool and Provider for GitHub Actions:

```bash
# Create Workload Identity Pool
gcloud iam workload-identity-pools create "github-pool" \
  --project="dislyze-staging2" \
  --location="global" \
  --display-name="GitHub Actions Pool"

# Create OIDC Provider
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
  --project="dislyze-staging2" \
  --location="global" \
  --workload-identity-pool="github-pool" \
  --display-name="GitHub Actions Provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.actor=assertion.actor" \
  --attribute-condition="assertion.repository_owner=='YOUR_GITHUB_USERNAME'" \
  --issuer-uri="https://token.actions.githubusercontent.com"

# Create service account for GitHub Actions
gcloud iam service-accounts create github-actions \
  --project="dislyze-staging2" \
  --display-name="GitHub Actions Service Account"

# Grant necessary permissions to the service account
gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/cloudsql.admin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/secretmanager.admin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.admin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountAdmin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/compute.viewer"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/serviceusage.serviceUsageAdmin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/resourcemanager.projectIamAdmin"

# Additional permissions for load balancer (Phase 4)
gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/compute.networkAdmin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/compute.securityAdmin"

gcloud projects add-iam-policy-binding dislyze-staging2 \
  --member="serviceAccount:github-actions@dislyze-staging2.iam.gserviceaccount.com" \
  --role="roles/certificatemanager.editor"

# Get your project number
gcloud projects describe dislyze-staging2 --format="value(projectNumber)"

# Allow the GitHub Actions workflow to impersonate the service account
# Replace YOUR_USERNAME with your GitHub username
gcloud iam service-accounts add-iam-policy-binding \
  github-actions@dislyze-staging2.iam.gserviceaccount.com \
  --project="dislyze-staging2" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/YOUR_USERNAME/dislyze"
```

## Step 4: Set up GitHub Repository Secrets

Add these secrets to your GitHub repository (Settings > Secrets and variables > Actions):

```
WIF_PROVIDER: projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/providers/github-provider
WIF_SERVICE_ACCOUNT: github-actions@dislyze-staging2.iam.gserviceaccount.com
PULUMI_ACCESS_TOKEN: your-pulumi-access-token
```

To get your project number:
```bash
gcloud projects describe dislyze-staging2 --format="value(projectNumber)"
```

## Step 5: Set up Pulumi

```bash
# Install Pulumi (if not already installed)
curl -fsSL https://get.pulumi.com | sh

# Login to Pulumi (you'll need a Pulumi account)
pulumi login

# Create a new access token at https://app.pulumi.com/account/tokens
# Add it as PULUMI_ACCESS_TOKEN in GitHub secrets
```

## Step 6: Deploy Infrastructure

```bash
# Navigate to infrastructure directory
cd infrastructure

# Install dependencies
npm install

# Create and select the staging stack
pulumi stack select staging --create

# Deploy the infrastructure
pulumi up
```

## Step 7: Update Database Password

After the infrastructure is deployed, update the database user password to match your secret:

```bash
# Get the actual password from Secret Manager
DB_PASSWORD=$(gcloud secrets versions access latest --secret="db-password")

# Update the database user password
gcloud sql users set-password dislyze \
  --instance=dislyze-staging-db \
  --password="$DB_PASSWORD"
```

## Step 8: Test the Deployment

After everything is set up, you can test by pushing to the `staging` branch or triggering the workflow manually.

The workflow will:
1. Build the Docker image
2. Push it to Artifact Registry
3. Deploy the infrastructure with Pulumi
4. Perform health checks

## Troubleshooting

### Common Issues

1. **Permission denied errors**: Ensure all IAM roles are properly assigned
2. **Secret not found**: Verify secrets are created in the correct project
3. **Docker build failures**: Check that all source files are correctly copied
4. **Database connection issues**: Verify the Cloud SQL instance is running and accessible

### Useful Commands

```bash
# Check Cloud Run service status
gcloud run services describe dislyze-staging-lugia --region=asia-northeast1

# View Cloud Run logs
gcloud run services logs read dislyze-staging-lugia --region=asia-northeast1

# List secrets
gcloud secrets list

# View Pulumi stack outputs
cd infrastructure && pulumi stack output
```

## Next Steps

Once Phase 1 is working:
1. Phase 2: Add frontend embedding
2. Phase 3: Add giratina-backend service
3. Phase 4: Add load balancer and custom domains
4. Phase 5: Add production environment
5. Phase 6: Add enterprise security features