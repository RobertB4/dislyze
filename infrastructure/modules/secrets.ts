import * as gcp from "@pulumi/gcp";

export interface SecretsInputs {
  apis: gcp.projects.Service[];
  secretsEncryptionKey: gcp.kms.CryptoKey;
  secretsKeyBinding: gcp.kms.CryptoKeyIAMBinding;
  region: string;
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
  const { apis, secretsEncryptionKey, secretsKeyBinding, region } = inputs;

  const dbPasswordSecret = new gcp.secretmanager.Secret(
    "db-password",
    {
      secretId: "db-password",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const lugiaAuthJwtSecret = new gcp.secretmanager.Secret(
    "lugia-auth-jwt-secret",
    {
      secretId: "lugia-auth-jwt-secret",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const giratinaAuthJwtSecret = new gcp.secretmanager.Secret(
    "giratina-auth-jwt-secret",
    {
      secretId: "giratina-auth-jwt-secret",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const createTenantJwtSecret = new gcp.secretmanager.Secret(
    "create-tenant-jwt-secret",
    {
      secretId: "create-tenant-jwt-secret",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const ipWhitelistEmergencyJwtSecret = new gcp.secretmanager.Secret(
    "ip-whitelist-emergency-jwt-secret",
    {
      secretId: "ip-whitelist-emergency-jwt-secret",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const initialPwSecret = new gcp.secretmanager.Secret(
    "initial-pw",
    {
      secretId: "initial-pw",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const internalUserPwSecret = new gcp.secretmanager.Secret(
    "internal-user-pw",
    {
      secretId: "internal-user-pw",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
  );

  const sendgridApiKeySecret = new gcp.secretmanager.Secret(
    "sendgrid-api-key",
    {
      secretId: "sendgrid-api-key",
      replication: {
        userManaged: {
          replicas: [{
            location: region,
            customerManagedEncryption: {
              kmsKeyName: secretsEncryptionKey.id,
            },
          }],
        },
      },
    },
    { dependsOn: [...apis, secretsEncryptionKey, secretsKeyBinding] }
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