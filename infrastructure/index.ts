import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

const config = new pulumi.Config();
const gcpConfig = new pulumi.Config("gcp");
const dbTier = config.require("db-tier");
const cloudRunCpu = config.require("cloudrun-cpu");
const cloudRunMemory = config.require("cloudrun-memory");
const cloudRunMaxInstances = config.require("cloudrun-max-instances");
const lugiaFrontendUrl = config.require("lugia-frontend-url");

export const projectId = gcpConfig.require("project");
export const region = gcpConfig.require("region");
export const environment = config.require("environment");

const enableApis = [
  "run.googleapis.com",
  "sql-component.googleapis.com",
  "sqladmin.googleapis.com",
  "secretmanager.googleapis.com",
  "artifactregistry.googleapis.com",
  "compute.googleapis.com",
];

const apis = enableApis.map(
  (api) =>
    // Pulumi resource names cannot contain dots
    new gcp.projects.Service(`enable-${api.replace(/\./g, "-")}`, {
      service: api,
      project: projectId,
      disableDependentServices: true,
    })
);

const artifactRegistry = new gcp.artifactregistry.Repository(
  "dislyze-docker-repo",
  {
    location: region,
    repositoryId: "dislyze",
    description: "Docker repository for Dislyze services",
    format: "DOCKER",
  },
  { dependsOn: apis }
);

// Secret Manager secrets
const dbPasswordSecret = new gcp.secretmanager.Secret(
  "db-password",
  {
    secretId: "db-password",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

const authJwtSecretSecret = new gcp.secretmanager.Secret(
  "auth-jwt-secret",
  {
    secretId: "auth-jwt-secret",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

const createTenantJwtSecretSecret = new gcp.secretmanager.Secret(
  "create-tenant-jwt-secret",
  {
    secretId: "create-tenant-jwt-secret",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

const ipWhitelistEmergencyJwtSecretSecret = new gcp.secretmanager.Secret(
  "ip-whitelist-emergency-jwt-secret",
  {
    secretId: "ip-whitelist-emergency-jwt-secret",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

const initialPwSecret = new gcp.secretmanager.Secret(
  "initial-pw",
  {
    secretId: "initial-pw",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

const internalUserPwSecret = new gcp.secretmanager.Secret(
  "internal-user-pw",
  {
    secretId: "internal-user-pw",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

const sendgridApiKeySecret = new gcp.secretmanager.Secret(
  "sendgrid-api-key",
  {
    secretId: "sendgrid-api-key",
    replication: {
      auto: {},
    },
  },
  { dependsOn: apis }
);

// Cloud SQL Database
const dbInstance = new gcp.sql.DatabaseInstance(
  "dislyze-db",
  {
    name: "dislyze-db",
    databaseVersion: "POSTGRES_17",
    region: region,
    deletionProtection: true,
    settings: {
      tier: dbTier,
      edition: "ENTERPRISE",
      availabilityType: "ZONAL", // Use REGIONAL for production
      backupConfiguration: {
        enabled: true,
        startTime: "03:00",
        location: region,
        transactionLogRetentionDays: 7,
        backupRetentionSettings: {
          retainedBackups: 7,
          retentionUnit: "COUNT",
        },
      },
      ipConfiguration: {
        ipv4Enabled: true,
        authorizedNetworks: [],
        sslMode: "ENCRYPTED_ONLY",
      },
      maintenanceWindow: {
        day: 7, // Sunday
        hour: 3,
        updateTrack: "stable",
      },
      databaseFlags: [
        {
          name: "max_connections",
          value: "100",
        },
      ],
    },
  },
  { dependsOn: apis }
);

const database = new gcp.sql.Database(
  "database",
  {
    name: "dislyze",
    instance: dbInstance.name,
  },
  { dependsOn: [dbInstance] }
);

const dbUser = new gcp.sql.User(
  "dislyze-db-user",
  {
    name: "dislyze-db-user",
    instance: dbInstance.name,
    password: "DEFAULT_PASSWORD", // Postgres requires a password. Actual value was updated after initial user creation.
  },
  { dependsOn: [dbInstance] }
);

// Service Account for Cloud Run
const cloudRunServiceAccount = new gcp.serviceaccount.Account(
  "cloudrun-sa",
  {
    accountId: "cloudrun-sa",
    displayName: "Cloud Run Service Account",
  },
  { dependsOn: apis }
);

const secretAccessorBinding = new gcp.projects.IAMMember(
  "secret-accessor",
  {
    project: projectId,
    role: "roles/secretmanager.secretAccessor",
    member: pulumi.interpolate`serviceAccount:${cloudRunServiceAccount.email}`,
  },
  { dependsOn: [cloudRunServiceAccount] }
);

const cloudSqlClientBinding = new gcp.projects.IAMMember(
  "cloudsql-client",
  {
    project: projectId,
    role: "roles/cloudsql.client",
    member: pulumi.interpolate`serviceAccount:${cloudRunServiceAccount.email}`,
  },
  { dependsOn: [cloudRunServiceAccount] }
);

// Get image tag from config, or fall back to currently deployed image
const lugiaImageTag = pulumi.all([region, projectId]).apply(async ([r, p]) => {
  if (config.get("lugia-image-tag")) {
    return config.get("lugia-image-tag");
  }

  try {
    const result = await gcp.cloudrun.getService({
      name: "lugia",
      location: r,
      project: p,
    });
    const image = result.templates?.[0]?.specs?.[0]?.containers?.[0]?.image;
    if (!image || !image.includes(":")) {
      return "latest";
    }
    return image.split(":")[1];
  } catch {
    return "latest";
  }
});

const lugiaService = new gcp.cloudrun.Service(
  "lugia",
  {
    name: "lugia",
    location: region,
    template: {
      metadata: {
        annotations: {
          "autoscaling.knative.dev/maxScale": cloudRunMaxInstances,
          "run.googleapis.com/cloudsql-instances": pulumi.interpolate`${projectId}:${region}:${dbInstance.name}`,
          "run.googleapis.com/client-name": "pulumi",
        },
      },
      spec: {
        serviceAccountName: cloudRunServiceAccount.email,
        timeoutSeconds: 60,
        containers: [
          {
            image: pulumi.interpolate`${region}-docker.pkg.dev/${projectId}/dislyze/lugia:${
              config.get("lugia-image-tag") || lugiaImageTag
            }`,
            resources: {
              limits: {
                cpu: cloudRunCpu,
                memory: cloudRunMemory,
              },
            },
            ports: [
              {
                containerPort: 8080,
              },
            ],
            envs: [
              {
                name: "APP_ENV",
                value: environment,
              },

              {
                name: "DB_HOST",
                value: pulumi.interpolate`/cloudsql/${projectId}:${region}:${dbInstance.name}`,
              },
              {
                name: "DB_USER",
                value: dbUser.name,
              },
              {
                name: "DB_PASSWORD",
                valueFrom: {
                  secretKeyRef: {
                    name: dbPasswordSecret.secretId,
                    key: "latest",
                  },
                },
              },
              {
                name: "DB_NAME",
                value: database.name,
              },
              {
                name: "DB_SSL_MODE",
                value: "require",
              },

              {
                name: "AUTH_JWT_SECRET",
                valueFrom: {
                  secretKeyRef: {
                    name: authJwtSecretSecret.secretId,
                    key: "latest",
                  },
                },
              },
              {
                name: "AUTH_RATE_LIMIT",
                value: "5",
              },
              {
                name: "CREATE_TENANT_JWT_SECRET",
                valueFrom: {
                  secretKeyRef: {
                    name: createTenantJwtSecretSecret.secretId,
                    key: "latest",
                  },
                },
              },
              {
                name: "IP_WHITELIST_EMERGENCY_JWT_SECRET",
                valueFrom: {
                  secretKeyRef: {
                    name: ipWhitelistEmergencyJwtSecretSecret.secretId,
                    key: "latest",
                  },
                },
              },

              {
                name: "SENDGRID_API_KEY",
                valueFrom: {
                  secretKeyRef: {
                    name: sendgridApiKeySecret.secretId,
                    key: "latest",
                  },
                },
              },
              {
                name: "SENDGRID_API_URL",
                value: "https://api.sendgrid.com/v3",
              },

              {
                name: "FRONTEND_URL",
                value: lugiaFrontendUrl,
              },

              {
                name: "INITIAL_PW",
                valueFrom: {
                  secretKeyRef: {
                    name: initialPwSecret.secretId,
                    key: "latest",
                  },
                },
              },
              {
                name: "INTERNAL_USER_PW",
                valueFrom: {
                  secretKeyRef: {
                    name: internalUserPwSecret.secretId,
                    key: "latest",
                  },
                },
              },
            ],
          },
        ],
      },
    },
    traffics: [
      {
        percent: 100,
        latestRevision: true,
      },
    ],
  },
  {
    dependsOn: [
      artifactRegistry,
      cloudRunServiceAccount,
      secretAccessorBinding,
      cloudSqlClientBinding,
      dbInstance,
      database,
      dbUser,
    ],
  }
);

// IAM policy to allow unauthenticated invocations (public access)
new gcp.cloudrun.IamPolicy(
  "lugia-iam",
  {
    project: projectId,
    location: region,
    service: lugiaService.name,
    policyData: JSON.stringify({
      bindings: [
        {
          role: "roles/run.invoker",
          members: ["allUsers"],
        },
      ],
    }),
  },
  { dependsOn: [lugiaService] }
);

export const artifactRegistryUrl = pulumi.interpolate`${region}-docker.pkg.dev/${projectId}/dislyze`;
export const databaseInstanceName = dbInstance.name;
export const databaseConnectionName = pulumi.interpolate`${projectId}:${region}:${dbInstance.name}`;
export const lugiaServiceUrl = lugiaService.statuses[0].url;
export const lugiaServiceName = lugiaService.name;
export const lugiaServiceResourceName = "lugia"; // For targeting in workflows
export const cloudRunServiceAccountEmail = cloudRunServiceAccount.email;
