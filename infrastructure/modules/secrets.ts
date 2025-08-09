import * as gcp from "@pulumi/gcp";

export interface SecretsInputs {
  apis: gcp.projects.Service[];
  secretsEncryptionKey: gcp.kms.CryptoKey;
}

export interface SecretsOutputs {
  dbPasswordSecret: gcp.secretmanager.Secret;
  lugiaAuthJwtSecret: gcp.secretmanager.Secret;
  giratinaAuthJwtSecret: gcp.secretmanager.Secret;
  createTenantJwtSecret: gcp.secretmanager.Secret;
  ipWhitelistEmergencyJwtSecret: gcp.secretmanager.Secret;
  initialPwSecret: gcp.secretmanager.Secret;
  internalUserPwSecret: gcp.secretmanager.Secret;
  sendgridApiKeySecret: gcp.secretmanager.Secret;
}

export function createSecrets(inputs: SecretsInputs): SecretsOutputs {
  const { apis, secretsEncryptionKey } = inputs;

  const dbPasswordSecret = new gcp.secretmanager.Secret(
    "db-password",
    {
      secretId: "db-password",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const lugiaAuthJwtSecret = new gcp.secretmanager.Secret(
    "lugia-auth-jwt-secret",
    {
      secretId: "lugia-auth-jwt-secret",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const giratinaAuthJwtSecret = new gcp.secretmanager.Secret(
    "giratina-auth-jwt-secret",
    {
      secretId: "giratina-auth-jwt-secret",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const createTenantJwtSecret = new gcp.secretmanager.Secret(
    "create-tenant-jwt-secret",
    {
      secretId: "create-tenant-jwt-secret",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const ipWhitelistEmergencyJwtSecret = new gcp.secretmanager.Secret(
    "ip-whitelist-emergency-jwt-secret",
    {
      secretId: "ip-whitelist-emergency-jwt-secret",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const initialPwSecret = new gcp.secretmanager.Secret(
    "initial-pw",
    {
      secretId: "initial-pw",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const internalUserPwSecret = new gcp.secretmanager.Secret(
    "internal-user-pw",
    {
      secretId: "internal-user-pw",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  const sendgridApiKeySecret = new gcp.secretmanager.Secret(
    "sendgrid-api-key",
    {
      secretId: "sendgrid-api-key",
      replication: {
        auto: {
          customerManagedEncryption: {
            kmsKeyName: secretsEncryptionKey.id,
          },
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey] }
  );

  return {
    dbPasswordSecret,
    lugiaAuthJwtSecret,
    giratinaAuthJwtSecret,
    createTenantJwtSecret,
    ipWhitelistEmergencyJwtSecret,
    initialPwSecret,
    internalUserPwSecret,
    sendgridApiKeySecret,
  };
}