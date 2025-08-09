import * as gcp from "@pulumi/gcp";

export interface KmsInputs {
  region: string;
  environment: string;
  apis: gcp.projects.Service[];
}

export interface KmsOutputs {
  keyRing: gcp.kms.KeyRing;
  databaseKey: gcp.kms.CryptoKey;
  secretsKey: gcp.kms.CryptoKey;
  auditLogsKey: gcp.kms.CryptoKey;
}

export function createKms(inputs: KmsInputs): KmsOutputs {
  const { region, environment, apis } = inputs;

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

  // NOTE: IAM bindings will be added in Phase 7 after service accounts are created
  // Service accounts don't exist until we first use Cloud SQL, Storage, and Secret Manager

  return {
    keyRing,
    databaseKey,
    secretsKey,
    auditLogsKey,
  };
}
