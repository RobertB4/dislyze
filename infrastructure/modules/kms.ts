import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface KmsInputs {
  projectId: string | pulumi.Output<string>;
  region: string;
  environment: string;
  apis: gcp.projects.Service[];
}

export interface KmsOutputs {
  keyRing: gcp.kms.KeyRing;
  databaseKey: gcp.kms.CryptoKey;
  secretsKey: gcp.kms.CryptoKey;
  auditLogsKey: gcp.kms.CryptoKey;
  // secretsKeyBinding: gcp.kms.CryptoKeyIAMBinding;
  // databaseKeyBinding: gcp.kms.CryptoKeyIAMBinding;
  // auditLogsKeyBinding: gcp.kms.CryptoKeyIAMBinding;
}

export function createKms(inputs: KmsInputs): KmsOutputs {
  const { projectId, region, environment, apis } = inputs;

  const keyRing = new gcp.kms.KeyRing(
    "dislyze-keyring",
    {
      name: `dislyze-keyring`,
      location: region,
    },
    { dependsOn: apis }
  );

  const databaseKey = new gcp.kms.CryptoKey(
    "database-key",
    {
      name: "database-encryption-key",
      keyRing: keyRing.id,
      purpose: "ENCRYPT_DECRYPT",
      rotationPeriod: "7776000s", // 90 days
      versionTemplate: {
        algorithm: "GOOGLE_SYMMETRIC_ENCRYPTION",
        protectionLevel: "SOFTWARE", // Can upgrade to HSM later if needed
      },
      labels: {
        purpose: "database-encryption",
        environment: environment,
        compliance: "iso-27001",
        "managed-by": "pulumi",
      },
    },
    { dependsOn: [keyRing] }
  );

  const secretsKey = new gcp.kms.CryptoKey(
    "secrets-key",
    {
      name: "secrets-encryption-key",
      keyRing: keyRing.id,
      purpose: "ENCRYPT_DECRYPT",
      rotationPeriod: "7776000s", // 90 days
      versionTemplate: {
        algorithm: "GOOGLE_SYMMETRIC_ENCRYPTION",
        protectionLevel: "SOFTWARE",
      },
      labels: {
        purpose: "secrets-encryption",
        environment: environment,
        compliance: "iso-27001",
        "managed-by": "pulumi",
      },
    },
    { dependsOn: [keyRing] }
  );

  // Audit logs encryption key (separate key for compliance auditing)
  const auditLogsKey = new gcp.kms.CryptoKey(
    "audit-logs-key",
    {
      name: "audit-logs-encryption-key",
      keyRing: keyRing.id,
      purpose: "ENCRYPT_DECRYPT",
      rotationPeriod: "7776000s", // 90 days
      versionTemplate: {
        algorithm: "GOOGLE_SYMMETRIC_ENCRYPTION",
        protectionLevel: "SOFTWARE",
      },
      labels: {
        purpose: "audit-logs-encryption",
        environment: environment,
        compliance: "iso-27001",
        "managed-by": "pulumi",
      },
    },
    { dependsOn: [keyRing] }
  );

  // Get project number for service account formatting
  const project = gcp.organizations.getProjectOutput({ projectId: projectId });
  project;

  // IAM bindings for service accounts to use the keys

  // Secret Manager service account binding for secrets key
  // const secretsKeyBinding = new gcp.kms.CryptoKeyIAMBinding(
  //   "secrets-key-sm-binding",
  //   {
  //     cryptoKeyId: secretsKey.id,
  //     role: "roles/cloudkms.cryptoKeyEncrypterDecrypter",
  //     members: [
  //       pulumi.interpolate`serviceAccount:service-${project.number}@gcp-sa-secretmanager.iam.gserviceaccount.com`,
  //     ],
  //   },
  //   { dependsOn: [secretsKey] }
  // );

  // Explicitly create the Cloud SQL service account
  const cloudSqlServiceIdentity = new gcp.projects.ServiceIdentity(
    "cloud-sql-service-identity",
    {
      service: "sqladmin.googleapis.com",
    },
    { dependsOn: apis }
  );
  cloudSqlServiceIdentity;

  // Cloud SQL service account binding for database key
  // const databaseKeyBinding = new gcp.kms.CryptoKeyIAMBinding(
  //   "database-key-sql-binding",
  //   {
  //     cryptoKeyId: databaseKey.id,
  //     role: "roles/cloudkms.cryptoKeyEncrypterDecrypter",
  //     members: [
  //       pulumi.interpolate`serviceAccount:${cloudSqlServiceIdentity.email}`,
  //     ],
  //   },
  //   { dependsOn: [databaseKey, cloudSqlServiceIdentity] }
  // );

  // Cloud Storage service account binding for audit logs key
  // const auditLogsKeyBinding = new gcp.kms.CryptoKeyIAMBinding(
  //   "audit-logs-key-gcs-binding",
  //   {
  //     cryptoKeyId: auditLogsKey.id,
  //     role: "roles/cloudkms.cryptoKeyEncrypterDecrypter",
  //     members: [
  //       pulumi.interpolate`serviceAccount:service-${project.number}@gs-project-accounts.iam.gserviceaccount.com`,
  //     ],
  //   },
  //   { dependsOn: [auditLogsKey] }
  // );

  return {
    keyRing,
    databaseKey,
    secretsKey,
    auditLogsKey,
    // secretsKeyBinding,
    // databaseKeyBinding,
    // auditLogsKeyBinding,
  };
}
