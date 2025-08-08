import * as pulumi from "@pulumi/pulumi";
import { createFoundation } from "./modules/foundation";
import { createSecrets } from "./modules/secrets";
import { createVpc } from "./modules/vpc";
import { createDatabase } from "./modules/database";
import { createServices } from "./modules/services";
import { createLoadBalancer } from "./modules/loadbalancer";
import { createMonitoring } from "./modules/monitoring";
import { createLogging } from "./modules/logging";
import { createCloudArmor } from "./modules/cloudarmor";

const config = new pulumi.Config();
const gcpConfig = new pulumi.Config("gcp");
const dbTier = config.require("db-tier");
const dbAvailabilityType = config.require("db-availability-type");
const cloudRunCpu = config.require("cloudrun-cpu");
const cloudRunMemory = config.require("cloudrun-memory");
const cloudRunMaxInstances = config.require("cloudrun-max-instances");
const lugiaFrontendUrl = config.require("lugia-frontend-url");
const giratinaFrontendUrl = config.require("giratina-frontend-url");
const lugiaDomain = config.require("lugia-domain");
const giratinaDomain = config.require("giratina-domain");
const alertEmail = config.get("alert-email") || "";

export const projectId = gcpConfig.require("project");
export const region = gcpConfig.require("region");
export const environment = config.require("environment");

const foundation = createFoundation({
  projectId,
  region,
});

const secrets = createSecrets({
  apis: foundation.apis,
});

const vpc = createVpc({
  projectId,
  region,
  apis: foundation.apis,
});

const db = createDatabase({
  projectId,
  region,
  dbTier,
  dbAvailabilityType,
  apis: foundation.apis,
  vpc: vpc.vpc,
  databaseSubnet: vpc.databaseSubnet,
});

const services = createServices({
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
  apis: foundation.apis,
  vpc: vpc.vpc,
  servicesSubnet: vpc.servicesSubnet,
  dbPasswordSecret: secrets.dbPasswordSecret,
  lugiaAuthJwtSecret: secrets.lugiaAuthJwtSecret,
  giratinaAuthJwtSecret: secrets.giratinaAuthJwtSecret,
  createTenantJwtSecret: secrets.createTenantJwtSecret,
  ipWhitelistEmergencyJwtSecret: secrets.ipWhitelistEmergencyJwtSecret,
  initialPwSecret: secrets.initialPwSecret,
  internalUserPwSecret: secrets.internalUserPwSecret,
  sendgridApiKeySecret: secrets.sendgridApiKeySecret,
});

const cloudArmor = createCloudArmor({
  projectId,
  environment,
  apis: foundation.apis,
});

const loadBalancer = createLoadBalancer({
  region,
  lugiaDomain,
  giratinaDomain,
  lugiaService: services.lugiaService,
  giratinaService: services.giratinaService,
  apis: foundation.apis,
  securityPolicy: cloudArmor.securityPolicy,
});

const logging = createLogging({
  projectId,
  environment,
});

const monitoring = createMonitoring({
  projectId,
  region,
  environment,
  alertEmail,
  lugiaDomain,
  giratinaDomain,
  lugiaService: services.lugiaService,
  giratinaService: services.giratinaService,
  dbInstance: db.dbInstance,
  apis: foundation.apis,
});

export const artifactRegistryUrl = pulumi.interpolate`${region}-docker.pkg.dev/${projectId}/dislyze`;

export const vpcName = vpc.vpc.name;
export const servicesSubnetName = vpc.servicesSubnet.name;
export const databaseSubnetName = vpc.databaseSubnet.name;

export const databaseInstanceName = db.databaseInstanceName;
export const databaseConnectionName = db.databaseConnectionName;

export const lugiaServiceUrl = services.lugiaServiceUrl;
export const lugiaServiceName = services.lugiaServiceName;
export const lugiaServiceResourceName = "lugia"; // For targeting in workflows
export const giratinaServiceUrl = services.giratinaServiceUrl;
export const giratinaServiceName = services.giratinaServiceName;
export const giratinaServiceResourceName = "giratina"; // For targeting in workflows
export const cloudRunServiceAccountEmail = services.cloudRunServiceAccountEmail;

export const auditLogBucket = logging.auditLogBucket.name;
export const auditLogSinks = {
  adminActivity: logging.adminActivitySink.name,
  applicationLogs: logging.auditLogSink.name,
};

export const loadBalancerIp = loadBalancer.loadBalancerIp;
export const lugiaUrl = `https://${lugiaDomain}`;
export const giratinaUrl = `https://${giratinaDomain}`;

export const securityPolicyName = cloudArmor.securityPolicy.name;

// Monitoring exports (only populated in production)
export const monitoringEnabled = environment === "production";
export const alertPoliciesCount = monitoring.alertPolicies?.length || 0;
