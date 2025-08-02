# Multi-Environment Deployment Plan for Dislyze

## Overview
Deploy dislyze application to Google Cloud with staging and production environments in separate GCP projects, using Workload Identity Federation for authentication, embedded frontends in Go binaries, Google Cloud Secret Manager, and GitHub Actions for CI/CD.

## Architecture Decisions

### Infrastructure
- **Two separate GCP projects**: `dislyze-staging` and `dislyze-production`
- **Region**: asia-northeast1 for all resources (unless GCP limitations require otherwise)
- **Authentication**: Workload Identity Federation (no service account keys)
- **Container Registry**: Google Artifact Registry
- **Secret Management**: Google Cloud Secret Manager (kebab-case naming)
- **Database**: Cloud SQL PostgreSQL, separate instances per environment
- **Scaling**: Max 1 instance per service, min 0 (serverless scaling)

### Application Architecture
- **Frontend Embedding**: Build frontends into Go binaries using `//go:embed`
- **Routing**: Load balancer routes to services based on domain
  - `app.mydomain.com` → lugia-backend (serves lugia-frontend for non-/api routes)
  - `internal.mydomain.com` → giratina-backend (serves giratina-frontend for non-/api routes)
- **Static File Serving**: Go backends serve embedded frontend files for all non-/api routes

### CI/CD Strategy
- **Branch-based deployment**:
  - `staging` branch push → deploy to `dislyze-staging` project
  - `main` branch push → deploy to `dislyze-production` project
- **Separate workflows**: deploy-lugia.yml and deploy-giratina.yml
- **Build process**: GitHub Actions → Build Docker images → Push to Artifact Registry → Deploy via Pulumi

### Domain
- **Our domain* We own the domain flownavi.com which we will use for this project
- **Lugia Service** This service is our main, user facing application.
  - **Lugia production** Will live under app.flownavi.com
  - **Lugia staging** Will live under staging-app.flownavi.com
- **Giratina Service** This service is our internal admin tool, used by employees of our company
  - **Giratina production** Will live under internal.flownavi.com
  - **Giratina staging** Will live under staging-internal.flownavi.com

## File Structure
```
dislyze/
├── .github/workflows/
│   ├── deploy-lugia.yml       # Lugia service deployment workflow
│   └── deploy-giratina.yml    # Giratina service deployment workflow
├── infrastructure/            # Pulumi infrastructure code
│   ├── index.ts              # Main infrastructure definition
│   ├── Pulumi.yaml           # Project configuration
│   ├── Pulumi.staging.yaml   # Staging stack config
│   └── Pulumi.production.yaml # Production stack config
├── lugia-backend/
│   ├── Dockerfile            # Multi-stage build with embedded frontend
│   └── [existing files]
├── giratina-backend/
│   ├── Dockerfile            # Multi-stage build with embedded frontend
│   └── [existing files]
└── docs/
    └── SUMMARY.md            # This deployment plan summary
```

## Implementation Phases (Incremental Approach)

> **Strategy**: Build incrementally with each phase resulting in a working system. This reduces debugging complexity and allows for early validation of each component.

### Phase 1: Minimal Viable Deployment (MVP)
**Objective**: Get lugia-backend API working in Cloud Run (staging only)

**Scope**: 
- Single GCP project (`dislyze-staging` only)
- Single service (lugia-backend API only, no frontend embedding)
- Basic infrastructure (Cloud SQL + Cloud Run)
- Essential authentication setup

**Detailed Tasks**:
1. **Setup Google Cloud Project**:
   ```bash
   gcloud projects create dislyze-staging
   gcloud config set project dislyze-staging
   gcloud services enable run.googleapis.com sql-component.googleapis.com secretmanager.googleapis.com artifactregistry.googleapis.com
   ```

2. **Create Artifact Registry**:
   ```bash
   gcloud artifacts repositories create dislyze --repository-format=docker --location=asia-northeast1
   ```

3. **Setup Workload Identity Federation**:
   ```bash
   # Create Workload Identity Pool
   gcloud iam workload-identity-pools create "github-pool" --location="global"
   # Create OIDC Provider
   gcloud iam workload-identity-pools providers create-oidc "github-provider" --workload-identity-pool="github-pool" --location="global" --issuer-uri="https://token.actions.githubusercontent.com" --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository"
   # Create service account and bind
   gcloud iam service-accounts create github-actions
   gcloud iam service-accounts add-iam-policy-binding github-actions@dislyze-staging.iam.gserviceaccount.com --role="roles/iam.workloadIdentityUser" --member="principalSet://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/YOUR_USERNAME/dislyze"
   ```

4. **Create Essential Secrets**:
   ```bash
   # You will run these with your actual secret values
   gcloud secrets create db-password --data-file=- <<< "YOUR_DB_PASSWORD"
   gcloud secrets create jwt-secret --data-file=- <<< "YOUR_JWT_SECRET"
   gcloud secrets create sendgrid-api-key --data-file=- <<< "YOUR_SENDGRID_KEY"
   ```

5. **Create Basic Pulumi Infrastructure**:
   - `infrastructure/index.ts` - Cloud SQL + single Cloud Run service
   - `infrastructure/Pulumi.staging.yaml` - Staging configuration
   - Basic Pulumi program focusing only on lugia-backend

6. **Create Simple Dockerfile**:
   - Basic Go build (no multi-stage, no frontend embedding)
   - Focus on getting the API working first

7. **Create Basic GitHub Actions Workflow**:
   - `deploy-lugia.yml` - Build, push, deploy lugia-backend only
   - Test with staging branch pushes

**Success Criteria**: 
- `curl https://CLOUD_RUN_URL/api/health` returns 200
- Database connection works
- Basic API endpoints respond correctly

**Deliverables**: Working lugia-backend API in Cloud Run

---

### Phase 2: Add Frontend Embedding
**Objective**: Full-stack lugia application with embedded frontend

**Detailed Tasks**:
1. **Update Dockerfile** to multi-stage build:
   ```dockerfile
   # Stage 1: Build frontend
   FROM node:18 AS frontend-builder
   WORKDIR /app
   COPY lugia-frontend/ ./
   RUN npm install && npm run build
   
   # Stage 2: Build Go backend with embedded frontend
   FROM golang:1.21 AS backend-builder
   WORKDIR /app
   COPY --from=frontend-builder /app/build ./frontend/dist
   COPY lugia-backend/ ./
   RUN go build -o main .
   
   # Stage 3: Final image
   FROM gcr.io/distroless/base
   COPY --from=backend-builder /app/main /main
   ENTRYPOINT ["/main"]
   ```

2. **Update Go Backend**:
   - Add `//go:embed frontend/dist/*` to embed static files
   - Add route handler for non-/api requests to serve frontend
   - Test SPA routing works correctly

3. **Test Full-Stack Application**:
   - Verify API endpoints still work
   - Verify frontend loads and routes correctly
   - Test frontend ↔ backend communication

**Success Criteria**: 
- Visit Cloud Run URL in browser, see working frontend
- Frontend can make API calls successfully
- SPA routing works (page refresh doesn't break)

**Deliverables**: Working full-stack lugia application

---

### Phase 3: Add Giratina Backend
**Objective**: Both services deployed independently

**Detailed Tasks**:
1. **Create Giratina Dockerfile** (copy pattern from lugia):
   - Multi-stage build with giratina-frontend embedding
   - Same structure as lugia Dockerfile

2. **Update Pulumi Infrastructure**:
   - Add second Cloud Run service for giratina-backend
   - Update IAM permissions for both services
   - Keep both services accessible via direct Cloud Run URLs

3. **Create Separate GitHub Actions Workflow**:
   - `deploy-giratina.yml` - Build and deploy giratina-backend
   - Test with staging branch pushes

4. **Update Go Backend for Giratina**:
   - Add static file serving similar to lugia
   - Ensure proper SPA routing

**Success Criteria**: 
- Both services accessible via their individual Cloud Run URLs
- Both frontends work independently
- Both APIs respond correctly

**Deliverables**: Two independent full-stack services

---

### Phase 4: Add Load Balancer & Custom Routing
**Objective**: Proper domain-based traffic routing

**Detailed Tasks**:
1. **Update Pulumi Infrastructure**:
   - Add Application Load Balancer
   - Configure URL maps for domain-based routing
   - Add Google-managed SSL certificates
   - Configure backend services pointing to Cloud Run

2. **Configure Routing Rules**:
   - `app.yourdomain.com` → lugia-backend
   - `internal.yourdomain.com` → giratina-backend
   - Handle both API and frontend requests properly

3. **Test Routing**:
   - Verify each domain routes to correct service
   - Verify SSL certificates work
   - Test both API and frontend through load balancer

**Success Criteria**: 
- Custom domains route correctly to respective services
- SSL works properly
- No degradation in functionality from previous phase

**Deliverables**: Properly routed multi-service application

---

### Phase 5: Add Production Environment
**Objective**: Two-environment setup with proper CI/CD

**Detailed Tasks**:
1. **Create Production GCP Project**:
   ```bash
   gcloud projects create dislyze-production
   # Repeat all setup steps from Phase 1 for production
   ```

2. **Duplicate Infrastructure**:
   - Create `Pulumi.production.yaml` config
   - Ensure same infrastructure pattern with production values
   - Create production secrets with secure values

3. **Update GitHub Actions**:
   - Modify workflows for branch-based deployment
   - `staging` branch → `dislyze-staging` project
   - `main` branch → `dislyze-production` project

4. **Test Environment Isolation**:
   - Deploy different versions to each environment
   - Verify complete isolation between environments
   - Test branch-based deployment flow

**Success Criteria**: 
- Independent staging and production environments
- Branch-based deployment works correctly
- No cross-environment interference

**Deliverables**: Complete two-environment setup

---

### Phase 6: Polish & Production Readiness
**Objective**: Production-ready system with enterprise security and monitoring

**Detailed Tasks**:
1. **Enterprise Security Hardening**:
   - Implement private VPC with controlled internet access
   - Add Cloud NAT for outbound traffic (no direct external IPs)
   - Enable comprehensive audit logging for all services
   - Implement customer-managed encryption keys (company-controlled)
   - Add container vulnerability scanning with deployment blocking
   - Configure Cloud Armor for DDoS protection and WAF rules

2. **Enterprise Monitoring & Alerting**:
   - **Security Monitoring**:
     - Failed authentication attempts (>10 in 1 minute)
     - Privilege escalation attempts
     - Unusual data access patterns
     - Geographic login anomalies
     - API rate limit violations
     - Suspicious file upload/download activity
   - **Compliance Monitoring**:
     - Data retention policy violations
     - Unencrypted data access attempts
     - Admin access without proper approval
     - Data export/backup failures
     - Audit log gaps or failures
   - **Basic Service Monitoring**:
     - Service health and uptime
     - API response times and error rates
     - Database connectivity and performance
     - Cost spike alerts

3. **Compliance Infrastructure**:
   - Configure log sinks for long-term audit storage
   - Implement Cloud Security Command Center integration
   - Add VPC Flow Logs for network monitoring
   - Set up automated security scanning and reporting

4. **Documentation & Runbooks**:
   - Create security incident response procedures
   - Document compliance monitoring processes
   - Add enterprise customer onboarding guides
   - Create monitoring and alerting playbooks

**Success Criteria**: 
- All enterprise security controls are active and monitored
- Security and compliance alerts trigger appropriately
- System meets SOC 2, ISO 27001, and GDPR requirements
- Audit logs are properly collected and stored
- Security incidents can be detected and responded to quickly

**Deliverables**: Enterprise-ready, compliant, and comprehensively monitored system

---

### Important
When building out every phase one by one, always keep in mind the end desired result at the end of phase 6.
This means we should make sure that we architect and design our code in a way that is compatible with what we want to achieve by the end of phase 6.

---

## Incremental Benefits

### Early Validation
- **Phase 1**: API works in cloud environment
- **Phase 2**: Full-stack deployment works
- **Phase 3**: Multi-service architecture works
- **Phase 4**: Load balancing and routing works
- **Phase 5**: Multi-environment CI/CD works

### Reduced Risk
- Each phase builds on proven foundation
- Issues isolated to single component changes
- Easy rollback to previous working state
- Incremental complexity increase

### Faster Time to Value
- Working system available after Phase 1
- Each phase adds functional value
- Can pause at any phase if needed
- Continuous progress visibility

## Security & Monitoring Requirements

### Enterprise Security Architecture
- **Private Network Infrastructure**: VPC with controlled internet access via Cloud NAT
- **Data Encryption**: Company-managed encryption keys with automatic rotation
- **Network Protection**: Cloud Armor for DDoS protection and WAF rules
- **Container Security**: Vulnerability scanning with deployment blocking for HIGH/CRITICAL issues
- **Audit Trail**: Comprehensive logging of all administrative and data access activities

### Authentication & Authorization
- **No long-lived credentials**: Workload Identity Federation eliminates service account keys
- **Principle of least privilege**: IAM permissions scoped to minimum required access
- **Secrets separation**: Sensitive data isolated in Secret Manager, regular config in Pulumi
- **Service account isolation**: Separate service accounts per service with minimal permissions

### Network Security
- **SSL/TLS termination**: All traffic encrypted in transit with Google-managed certificates
- **Private database access**: Cloud SQL instances accessible only through private network
- **VPC isolation**: Complete network separation between environments
- **Flow logging**: VPC Flow Logs for network traffic monitoring and analysis

### Data Protection
- **Encryption at Rest**: All data encrypted using company-managed KMS keys
- **Encryption in Transit**: TLS 1.3 for all API communications
- **Key Rotation**: Automatic encryption key rotation every 30 days
- **Backup Security**: Encrypted backups with separate access controls

### Compliance & Monitoring
- **Audit Logging**: Comprehensive audit trails for all system activities
- **Log Retention**: Long-term storage of audit logs for compliance requirements
- **Security Monitoring**: Real-time detection of security events and anomalies
- **Compliance Reporting**: Automated compliance status reporting and alerting

### Monitoring Categories

#### Security Monitoring (Real-time Alerts)
- Failed authentication attempts (>10 in 1 minute)
- Privilege escalation attempts
- Unusual data access patterns
- Geographic login anomalies (new countries/regions)
- API rate limit violations
- Suspicious file upload/download activity

#### Compliance Monitoring (Automated Checks)
- Data retention policy violations
- Unencrypted data access attempts
- Administrative access without proper approval workflow
- Data export/backup operation failures
- Audit log collection gaps or failures

#### Operational Monitoring (Service Health)
- Service availability and response times
- Database connectivity and performance metrics
- Error rates and exception tracking
- Resource utilization and cost monitoring
- Deployment success/failure tracking

### Compliance Certifications Target
- **SOC 2 Type II**: Security, availability, and confidentiality controls
- **ISO 27001**: Information security management system
- **GDPR**: Data protection and privacy compliance
- **Enterprise Ready**: Meets big enterprise security requirements

## Operational Benefits

### Deployment
- **Atomic deployments**: Frontend and backend always deployed together and in sync
- **Zero-downtime deployments**: Cloud Run handles traffic shifting automatically
- **Rollback capability**: Pulumi state enables easy rollbacks to previous versions

### Scaling & Performance
- **Serverless scaling**: Cloud Run scales from 0 to 1 instances based on traffic
- **Cost optimization**: Pay only for actual usage, not idle resources
- **Regional deployment**: asia-northeast1 provides low latency for target users

### Maintenance
- **Infrastructure as Code**: All infrastructure changes version controlled
- **Environment parity**: Staging mirrors production for reliable testing
- **Centralized monitoring**: Google Cloud's built-in observability tools

## Cost Estimation

### Basic Usage (Phase 1-5)
- **Cloud Run**: $0-20/month (serverless pricing, pay per request)
- **Cloud SQL**: $25-35/month (db-f1-micro instances)
- **Load Balancer**: $18/month (Application Load Balancer)
- **Storage**: $5-10/month (container images, logs)
- **Secret Manager**: $1-2/month (8 secrets × 2 environments)

**Total estimated cost**: $49-85/month for both staging and production environments

### Enterprise Security & Monitoring (Phase 6 additions)
- **Cloud KMS**: $1/month per key + usage (~$3-5/month total)
- **Cloud Armor**: $5/month + per-request fees
- **VPC Flow Logs**: $0.50 per GB of logs (~$10-20/month)
- **Audit log storage**: $0.02 per GB/month (~$5-15/month)
- **Additional monitoring**: $10-20/month (dashboards, alerting)

**Additional enterprise cost**: $33-65/month

**Total with enterprise features**: $82-150/month for both environments

### Scaling Considerations
- Costs scale linearly with usage (requests, database load)
- Can upgrade database instances as needed
- Load balancer supports high traffic without additional costs
- Container image storage costs negligible for typical usage

## Next Steps

1. **Confirm domain setup**: Decide on actual domain names for routing configuration
2. **Review security requirements**: Ensure compliance with any specific security policies
3. **Set monitoring alerts**: Configure alerting for system health and performance

This plan provides a robust, scalable, and secure foundation for deploying the dislyze application to Google Cloud with proper multi-environment support.