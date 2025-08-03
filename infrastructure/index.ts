import * as pulumi from "@pulumi/pulumi";
import { createFoundation } from "./modules/foundation";
import { createSecrets } from "./modules/secrets";
import { createNetworking } from "./modules/networking";
import { createDatabase } from "./modules/database";
import { createServices } from "./modules/services";

const config = new pulumi.Config();
const gcpConfig = new pulumi.Config("gcp");
const dbTier = config.require("db-tier");
const cloudRunCpu = config.require("cloudrun-cpu");
const cloudRunMemory = config.require("cloudrun-memory");
const cloudRunMaxInstances = config.require("cloudrun-max-instances");
const lugiaFrontendUrl = config.require("lugia-frontend-url");
const giratinaFrontendUrl = config.require("giratina-frontend-url");
const lugiaDomain = config.require("lugia-domain");
const giratinaDomain = config.require("giratina-domain");

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

const db = createDatabase({
  projectId,
  region,
  dbTier,
  apis: foundation.apis,
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
  dbPasswordSecret: secrets.dbPasswordSecret,
  lugiaAuthJwtSecret: secrets.lugiaAuthJwtSecret,
  giratinaAuthJwtSecret: secrets.giratinaAuthJwtSecret,
  createTenantJwtSecret: secrets.createTenantJwtSecret,
  ipWhitelistEmergencyJwtSecret: secrets.ipWhitelistEmergencyJwtSecret,
  initialPwSecret: secrets.initialPwSecret,
  internalUserPwSecret: secrets.internalUserPwSecret,
  sendgridApiKeySecret: secrets.sendgridApiKeySecret,
});

const networking = createNetworking({
  projectId,
  region,
  lugiaDomain,
  giratinaDomain,
  lugiaService: services.lugiaService,
  giratinaService: services.giratinaService,
  apis: foundation.apis,
});

export const artifactRegistry = foundation.artifactRegistry;
export const artifactRegistryUrl = pulumi.interpolate`${region}-docker.pkg.dev/${projectId}/dislyze`;
export const databaseInstanceName = db.databaseInstanceName;
export const databaseConnectionName = db.databaseConnectionName;
export const lugiaServiceUrl = services.lugiaServiceUrl;
export const lugiaServiceName = services.lugiaServiceName;
export const lugiaServiceResourceName = "lugia"; // For targeting in workflows
export const giratinaServiceUrl = services.giratinaServiceUrl;
export const giratinaServiceName = services.giratinaServiceName;
export const giratinaServiceResourceName = "giratina"; // For targeting in workflows
export const cloudRunServiceAccountEmail = services.cloudRunServiceAccountEmail;

export const loadBalancerIp = networking.loadBalancerIp;

export const lugiaUrl = `https://${lugiaDomain}`;
export const giratinaUrl = `https://${giratinaDomain}`;
