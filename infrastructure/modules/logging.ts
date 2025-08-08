import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface LoggingInputs {
  projectId: string | pulumi.Output<string>;
  environment: string;
}

export interface LoggingOutputs {
  auditLogBucket: gcp.storage.Bucket;
  auditLogSink: gcp.logging.ProjectSink;
  adminActivitySink: gcp.logging.ProjectSink;
}

export function createLogging(inputs: LoggingInputs): LoggingOutputs {
  const { projectId, environment } = inputs;

  // Create Cloud Storage bucket for long-term audit log storage
  const auditLogBucket = new gcp.storage.Bucket(
    "audit-logs-bucket",
    {
      name: pulumi.interpolate`${projectId}-audit-logs-${environment}`,
      location: "ASIA-NORTHEAST1",
      // 1 year retention for audit logs
      lifecycleRules: [
        {
          condition: {
            age: 365, // 1 year in days
          },
          action: {
            type: "Delete",
          },
        },
      ],
      // Prevent accidental deletion
      retentionPolicy: {
        retentionPeriod: 365 * 24 * 60 * 60, // 1 year in seconds
      },
      // Enable versioning for audit trail integrity
      versioning: {
        enabled: true,
      },
      // Uniform bucket-level access for security
      uniformBucketLevelAccess: true,
      // Server-side encryption
      encryption: {
        defaultKmsKeyName: "", // Will use Google-managed keys for now
      },
    }
  );

  // Admin Activity Audit Log Sink (most critical events)
  const adminActivitySink = new gcp.logging.ProjectSink(
    "admin-activity-audit-sink",
    {
      name: "admin-activity-audit-sink",
      destination: pulumi.interpolate`storage.googleapis.com/${auditLogBucket.name}/admin-activity`,
      filter: `
        protoPayload.serviceName="cloudresourcemanager.googleapis.com" OR
        protoPayload.serviceName="iam.googleapis.com" OR
        protoPayload.serviceName="cloudsql.googleapis.com" OR
        protoPayload.serviceName="run.googleapis.com" OR
        protoPayload.serviceName="secretmanager.googleapis.com" OR
        protoPayload.serviceName="compute.googleapis.com"
      `.replace(/\s+/g, ' ').trim(),
      description: "Sink for admin activity audit logs to Cloud Storage",
    },
    { dependsOn: [auditLogBucket] }
  );

  // IAM member for admin activity sink to write to bucket
  new gcp.storage.BucketIAMMember("admin-activity-bucket-writer", {
    bucket: auditLogBucket.name,
    role: "roles/storage.objectCreator",
    member: adminActivitySink.writerIdentity,
  }, { dependsOn: [adminActivitySink] });


  // Application Log Sink (Cloud Run application logs)
  const auditLogSink = new gcp.logging.ProjectSink(
    "application-audit-sink",
    {
      name: "application-audit-sink", 
      destination: pulumi.interpolate`storage.googleapis.com/${auditLogBucket.name}/application-logs`,
      filter: `
        resource.type="cloud_run_revision" AND
        (severity="ERROR" OR
         severity="WARNING" OR
         severity="CRITICAL" OR
         jsonPayload.event_type:"auth_failure" OR
         jsonPayload.event_type:"auth_success" OR
         textPayload:"[AUTH]")
      `.replace(/\s+/g, ' ').trim(),
      description: "Sink for Cloud Run application audit events",
    },
    { dependsOn: [auditLogBucket] }
  );

  // IAM member for application audit sink to write to bucket
  new gcp.storage.BucketIAMMember("application-audit-bucket-writer", {
    bucket: auditLogBucket.name,
    role: "roles/storage.objectCreator",
    member: auditLogSink.writerIdentity,
  }, { dependsOn: [auditLogSink] });

  return {
    auditLogBucket,
    auditLogSink,
    adminActivitySink,
  };
}