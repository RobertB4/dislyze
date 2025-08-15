import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface GitHubActionsIAMInputs {
  projectId: string | pulumi.Output<string>;
  serviceAccountEmail: string | pulumi.Output<string>;
}

export interface GitHubActionsIAMOutputs {
  projectIAMBindings: gcp.projects.IAMMember[];
}

export function createGitHubActionsIAM(
  inputs: GitHubActionsIAMInputs
): GitHubActionsIAMOutputs {
  const { projectId, serviceAccountEmail } = inputs;

  // Define exactly the same roles as in setup.md (in the same order)
  const requiredRoles = [
    "roles/run.admin",
    "roles/cloudsql.admin",
    "roles/secretmanager.admin",
    "roles/artifactregistry.admin",
    "roles/iam.serviceAccountAdmin",
    "roles/iam.serviceAccountUser",
    "roles/compute.viewer",
    "roles/serviceusage.serviceUsageAdmin",
    "roles/resourcemanager.projectIamAdmin",
    "roles/compute.networkAdmin",
    "roles/compute.securityAdmin",
    "roles/compute.loadBalancerAdmin",
    "roles/certificatemanager.editor",
    "roles/monitoring.admin",
    "roles/vpcaccess.admin",
    "roles/ondemandscanning.admin",
    "roles/containeranalysis.admin",
    "roles/logging.admin",
  ];

  // Create project IAM members (equivalent to gcloud projects add-iam-policy-binding)
  const projectIAMBindings = requiredRoles.map(
    (role) =>
      new gcp.projects.IAMMember(
        `github-actions-${role.replace(/[.\/]/g, "-")}`,
        {
          project: projectId,
          role: role,
          member: pulumi.interpolate`serviceAccount:${serviceAccountEmail}`,
        }
      )
  );

  return {
    projectIAMBindings,
  };
}
