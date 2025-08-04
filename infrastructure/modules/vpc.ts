import * as gcp from "@pulumi/gcp";
import { Firewall } from "@pulumi/gcp/compute";
import * as pulumi from "@pulumi/pulumi";

export interface VpcInputs {
  projectId: string | pulumi.Output<string>;
  region: string | pulumi.Output<string>;
  apis: gcp.projects.Service[];
}

export interface VpcOutputs {
  vpc: gcp.compute.Network;
  servicesSubnet: gcp.compute.Subnetwork;
  databaseSubnet: gcp.compute.Subnetwork;
  router: gcp.compute.Router;
  nat: gcp.compute.RouterNat;
  dbAccessFirewall: Firewall;
  healthCheckFirewall: Firewall;
}

export function createVpc(inputs: VpcInputs): VpcOutputs {
  const { region, apis } = inputs;

  const vpc = new gcp.compute.Network(
    "dislyze-vpc",
    {
      name: "dislyze-vpc",
      autoCreateSubnetworks: false,
      routingMode: "REGIONAL",
    },
    { dependsOn: apis }
  );

  const servicesSubnet = new gcp.compute.Subnetwork(
    "services-subnet",
    {
      name: "services-subnet",
      ipCidrRange: "10.0.1.0/24",
      region: region,
      network: vpc.id,
      privateIpGoogleAccess: true,
    },
    { dependsOn: vpc }
  );

  const databaseSubnet = new gcp.compute.Subnetwork(
    "database-subnet",
    {
      name: "database-subnet",
      ipCidrRange: "10.0.2.0/28",
      region: region,
      network: vpc.id,
      privateIpGoogleAccess: true,
    },
    { dependsOn: vpc }
  );

  const router = new gcp.compute.Router(
    "dislyze-router",
    {
      name: "dislyze-router",
      region: region,
      network: vpc.id,
    },
    { dependsOn: vpc }
  );

  const nat = new gcp.compute.RouterNat(
    "dislyze-nat",
    {
      name: "dislyze-nat",
      router: router.name,
      region: region,
      natIpAllocateOption: "AUTO_ONLY",
      sourceSubnetworkIpRangesToNat: "ALL_SUBNETWORKS_ALL_IP_RANGES",
    },
    { dependsOn: router }
  );

  const dbAccessFirewall = new gcp.compute.Firewall(
    "allow-services-to-db",
    {
      name: "allow-services-to-db",
      network: vpc.id,
      allows: [
        {
          protocol: "tcp",
          ports: ["5432"], // PostgreSQL port
        },
      ],
      sourceRanges: ["10.0.1.0/24"], // Services subnet
      targetTags: ["database"],
    },
    { dependsOn: vpc }
  );

  const healthCheckFirewall = new gcp.compute.Firewall(
    "allow-health-checks",
    {
      name: "allow-health-checks",
      network: vpc.id,
      allows: [
        {
          protocol: "tcp",
          ports: ["8080"], // Cloud Run default port
        },
      ],
      sourceRanges: [
        "130.211.0.0/22", // Google health check ranges
        "35.191.0.0/16",
      ],
      targetTags: ["cloud-run"],
    },
    { dependsOn: vpc }
  );

  return {
    vpc,
    servicesSubnet,
    databaseSubnet,
    router,
    nat,
    dbAccessFirewall,
    healthCheckFirewall,
  };
}
