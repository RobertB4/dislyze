import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface FoundationInputs {
  projectId: string | pulumi.Output<string>;
  region: string | pulumi.Output<string>;
}

export interface FoundationOutputs {
  apis: gcp.projects.Service[];
  artifactRegistry: gcp.artifactregistry.Repository;
}

export function createFoundation(inputs: FoundationInputs): FoundationOutputs {
  const { projectId, region } = inputs;

  const enableApis = [
    "run.googleapis.com",
    "sql-component.googleapis.com",
    "sqladmin.googleapis.com",
    "secretmanager.googleapis.com",
    "artifactregistry.googleapis.com",
    "compute.googleapis.com",
    "certificatemanager.googleapis.com",
    "monitoring.googleapis.com",
    "logging.googleapis.com",
    "vpcaccess.googleapis.com",
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

  return {
    apis,
    artifactRegistry,
  };
}