# Infrastructure Setup Guide

This guide will help you set up the Google Cloud infrastructure and secrets for the Dislyze deployment.

## Prerequisites

- Google Cloud SDK (`gcloud`) installed and configured
- Pulumi CLI installed
- Node.js 20+ installed
- Access to the `{project_id}` Google Cloud project

## Step 1: Set up Google Cloud Configuration

# Ensure you're authenticated
gcloud auth login

```bash
# Set the project
gcloud config set project {project_id}

# Enable required APIs - may enable one by one if the command fails
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

**Important**: Replace the placeholder values with your actual secrets!

## Step 2: Set up Workload Identity Federation

Create a Workload Identity Pool and Provider for GitHub Actions:

```bash
# Create Workload Identity Pool
gcloud iam workload-identity-pools create "github-pool" \
  --project="{project_id}" \
  --location="global" \
  --display-name="GitHub Actions Pool"

# Create OIDC Provider
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
  --project="{project_id}" \
  --location="global" \
  --workload-identity-pool="github-pool" \
  --display-name="GitHub Actions Provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.actor=assertion.actor" \
  --attribute-condition="assertion.repository_owner=='YOUR_GITHUB_USERNAME'" \
  --issuer-uri="https://token.actions.githubusercontent.com"

# Create service account for GitHub Actions
gcloud iam service-accounts create github-actions \
  --project="{project_id}" \
  --display-name="GitHub Actions Service Account"

gcloud projects add-iam-policy-binding {project_id} \
  --member="serviceAccount:github-actions@{project_id}.iam.gserviceaccount.com" \
  --role="roles/resourcemanager.projectIamAdmin"

gcloud projects add-iam-policy-binding {project_id} \
  --member="serviceAccount:github-actions@{project_id}.iam.gserviceaccount.com" \
  --role="roles/monitoring.admin"

  gcloud projects add-iam-policy-binding {project_id} \
  --member="serviceAccount:github-actions@{project_id}.iam.gserviceaccount.com" \
  --role="roles/run.admin"

  gcloud projects add-iam-policy-binding {project_id} \
  --member="serviceAccount:github-actions@{project_id}.iam.gserviceaccount.com" \
  --role="roles/compute.viewer"

# Get your project number
gcloud projects describe {project_id} --format="value(projectNumber)"

# Allow the GitHub Actions workflow to impersonate the service account
# Replace YOUR_USERNAME with your GitHub username
gcloud iam service-accounts add-iam-policy-binding \
  github-actions@{project_id}.iam.gserviceaccount.com \
  --project="{project_id}" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/YOUR_USERNAME/dislyze"
```

## Step 3: Set up GitHub Repository Secrets

Add these secrets to your GitHub repository (Settings > Secrets and variables > Actions):

```
WIF_PROVIDER: projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/providers/github-provider
WIF_SERVICE_ACCOUNT: github-actions@{project_id}.iam.gserviceaccount.com
PULUMI_ACCESS_TOKEN: your-pulumi-access-token
```

Also set up GitHub enironment variables (not secret)

```
PROJECT_ID: {project_id}
```

To get your project number:
```bash
gcloud projects describe {project_id} --format="value(projectNumber)"
```

## Step 4: Set up Pulumi

```bash
# Install Pulumi (if not already installed)
curl -fsSL https://get.pulumi.com | sh

# Login to Pulumi (you'll need a Pulumi account)
pulumi login

# Create a new access token at https://app.pulumi.com/account/tokens
# Add it as PULUMI_ACCESS_TOKEN in GitHub secrets

Create/pdate infrastructure/Pulumi.{environment}.stack file
```

## Step 5: Deploy Infrastructure

Remove everything but createGitHubActionsIAM from index.ts
Then slowly add more on every deploy until everything succeeds

Run github actions

## Step 6: Update Database Password

After the infrastructure is deployed, update the database user password to match your secret:

```bash
# Get the actual password from Secret Manager
DB_PASSWORD=$(gcloud secrets versions access latest --secret="db-password")

# Update the database user password
gcloud sql users set-password dislyze \
  --instance=dislyze-staging-db \
  --password="$DB_PASSWORD"
```

## Step 7: Update Secret Manager Secrets

You'll need to update all secrets with your actual values