import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface DatabaseInputs {
  projectId: string | pulumi.Output<string>;
  region: string | pulumi.Output<string>;
  dbTier: string;
  apis: gcp.projects.Service[];
}

export interface DatabaseOutputs {
  dbInstance: gcp.sql.DatabaseInstance;
  database: gcp.sql.Database;
  dbUser: gcp.sql.User;
  databaseInstanceName: pulumi.Output<string>;
  databaseConnectionName: pulumi.Output<string>;
}

export function createDatabase(inputs: DatabaseInputs): DatabaseOutputs {
  const { projectId, region, dbTier, apis } = inputs;

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

  return {
    dbInstance,
    database,
    dbUser,
    databaseInstanceName: dbInstance.name,
    databaseConnectionName: pulumi.interpolate`${projectId}:${region}:${dbInstance.name}`,
  };
}
