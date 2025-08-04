import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";
import { DatabaseOutputs } from "./database";

export interface ServicesInputs {
  projectId: string | pulumi.Output<string>;
  region: string | pulumi.Output<string>;
  environment: string;
  cloudRunCpu: string;
  cloudRunMemory: string;
  cloudRunMaxInstances: string;
  lugiaFrontendUrl: string;
  giratinaFrontendUrl: string;
  config: pulumi.Config;
  db: DatabaseOutputs;
  apis: gcp.projects.Service[];
  vpc: gcp.compute.Network;
  servicesSubnet: gcp.compute.Subnetwork;
  dbPasswordSecret: gcp.secretmanager.Secret;
  lugiaAuthJwtSecret: gcp.secretmanager.Secret;
  giratinaAuthJwtSecret: gcp.secretmanager.Secret;
  createTenantJwtSecret: gcp.secretmanager.Secret;
  ipWhitelistEmergencyJwtSecret: gcp.secretmanager.Secret;
  initialPwSecret: gcp.secretmanager.Secret;
  internalUserPwSecret: gcp.secretmanager.Secret;
  sendgridApiKeySecret: gcp.secretmanager.Secret;
}

export interface ServicesOutputs {
  cloudRunServiceAccount: gcp.serviceaccount.Account;
  lugiaService: gcp.cloudrun.Service;
  giratinaService: gcp.cloudrun.Service;
  lugiaServiceUrl: pulumi.Output<string>;
  lugiaServiceName: pulumi.Output<string>;
  giratinaServiceUrl: pulumi.Output<string>;
  giratinaServiceName: pulumi.Output<string>;
  cloudRunServiceAccountEmail: pulumi.Output<string>;
}

export function createServices(inputs: ServicesInputs): ServicesOutputs {
  const {
    projectId,
    region,
    environment,
    cloudRunCpu,
    cloudRunMemory,
    cloudRunMaxInstances,
    lugiaFrontendUrl,
    giratinaFrontendUrl,
    config,
    db,
    apis,
    vpc,
    dbPasswordSecret,
    lugiaAuthJwtSecret,
    giratinaAuthJwtSecret,
    createTenantJwtSecret,
    ipWhitelistEmergencyJwtSecret,
    initialPwSecret,
    internalUserPwSecret,
    sendgridApiKeySecret,
  } = inputs;

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

  const lugiaImageTag = pulumi
    .all([region, projectId])
    .apply(async ([r, p]) => {
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

  const vpcConnector = new gcp.vpcaccess.Connector(
    "cloudrun-vpc-connector",
    {
      name: "cloudrun-vpc-connector",
      region: region,
      network: vpc.name,
      ipCidrRange: "10.0.3.0/28",
      minThroughput: 200,
      maxThroughput: 300,
    },
    { dependsOn: [...apis, vpc] }
  );

  const lugiaService = new gcp.cloudrun.Service(
    "lugia",
    {
      name: "lugia",
      location: region,
      template: {
        metadata: {
          annotations: {
            "autoscaling.knative.dev/maxScale": cloudRunMaxInstances,
            "run.googleapis.com/cloudsql-instances": db.databaseConnectionName,
            "run.googleapis.com/client-name": "pulumi",
            "run.googleapis.com/vpc-access-connector": vpcConnector.name,
            "run.googleapis.com/vpc-access-egress": "private-ranges-only",
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
                  value: pulumi.interpolate`/cloudsql/${db.databaseConnectionName}`,
                },
                {
                  name: "DB_USER",
                  value: db.dbUser.name,
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
                  value: db.database.name,
                },
                {
                  name: "DB_SSL_MODE",
                  value: "require",
                },
                {
                  name: "AUTH_JWT_SECRET",
                  valueFrom: {
                    secretKeyRef: {
                      name: lugiaAuthJwtSecret.secretId,
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
                      name: createTenantJwtSecret.secretId,
                      key: "latest",
                    },
                  },
                },
                {
                  name: "IP_WHITELIST_EMERGENCY_JWT_SECRET",
                  valueFrom: {
                    secretKeyRef: {
                      name: ipWhitelistEmergencyJwtSecret.secretId,
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
        cloudRunServiceAccount,
        secretAccessorBinding,
        cloudSqlClientBinding,
        db.dbInstance,
        db.database,
        db.dbUser,
      ],
    }
  );

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

  const giratinaImageTag = pulumi
    .all([region, projectId])
    .apply(async ([r, p]) => {
      if (config.get("giratina-image-tag")) {
        return config.get("giratina-image-tag");
      }

      try {
        const result = await gcp.cloudrun.getService({
          name: "giratina",
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

  const giratinaService = new gcp.cloudrun.Service(
    "giratina",
    {
      name: "giratina",
      location: region,
      template: {
        metadata: {
          annotations: {
            "autoscaling.knative.dev/maxScale": cloudRunMaxInstances,
            "run.googleapis.com/cloudsql-instances": db.databaseConnectionName,
            "run.googleapis.com/client-name": "pulumi",
            "run.googleapis.com/vpc-access-connector": vpcConnector.name,
            "run.googleapis.com/vpc-access-egress": "private-ranges-only",
          },
        },
        spec: {
          serviceAccountName: cloudRunServiceAccount.email,
          timeoutSeconds: 60,
          containers: [
            {
              image: pulumi.interpolate`${region}-docker.pkg.dev/${projectId}/dislyze/giratina:${
                config.get("giratina-image-tag") || giratinaImageTag
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
                  value: pulumi.interpolate`/cloudsql/${db.databaseConnectionName}`,
                },
                {
                  name: "DB_USER",
                  value: db.dbUser.name,
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
                  value: db.database.name,
                },
                {
                  name: "DB_SSL_MODE",
                  value: "require",
                },
                {
                  name: "AUTH_JWT_SECRET",
                  valueFrom: {
                    secretKeyRef: {
                      name: giratinaAuthJwtSecret.secretId,
                      key: "latest",
                    },
                  },
                },
                {
                  name: "AUTH_RATE_LIMIT",
                  value: "5",
                },
                {
                  name: "LUGIA_AUTH_JWT_SECRET",
                  valueFrom: {
                    secretKeyRef: {
                      name: lugiaAuthJwtSecret.secretId,
                      key: "latest",
                    },
                  },
                },
                {
                  name: "CREATE_TENANT_JWT_SECRET",
                  valueFrom: {
                    secretKeyRef: {
                      name: createTenantJwtSecret.secretId,
                      key: "latest",
                    },
                  },
                },
                {
                  name: "IP_WHITELIST_EMERGENCY_JWT_SECRET",
                  valueFrom: {
                    secretKeyRef: {
                      name: ipWhitelistEmergencyJwtSecret.secretId,
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
                  value: giratinaFrontendUrl,
                },
                {
                  name: "LUGIA_FRONTEND_URL",
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
        cloudRunServiceAccount,
        secretAccessorBinding,
        cloudSqlClientBinding,
        db.dbInstance,
        db.database,
        db.dbUser,
      ],
    }
  );

  new gcp.cloudrun.IamPolicy(
    "giratina-iam",
    {
      project: projectId,
      location: region,
      service: giratinaService.name,
      policyData: JSON.stringify({
        bindings: [
          {
            role: "roles/run.invoker",
            members: ["allUsers"],
          },
        ],
      }),
    },
    { dependsOn: [giratinaService] }
  );

  return {
    cloudRunServiceAccount,
    lugiaService,
    giratinaService,
    lugiaServiceUrl: lugiaService.statuses[0].url,
    lugiaServiceName: lugiaService.name,
    giratinaServiceUrl: giratinaService.statuses[0].url,
    giratinaServiceName: giratinaService.name,
    cloudRunServiceAccountEmail: cloudRunServiceAccount.email,
  };
}
