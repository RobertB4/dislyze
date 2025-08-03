import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface NetworkingInputs {
  projectId: string | pulumi.Output<string>;
  region: string | pulumi.Output<string>;
  lugiaDomain: string;
  giratinaDomain: string;
  lugiaService: gcp.cloudrun.Service;
  giratinaService: gcp.cloudrun.Service;
  apis: gcp.projects.Service[];
}

export interface NetworkingOutputs {
  staticIp: gcp.compute.GlobalAddress;
  loadBalancerIp: pulumi.Output<string>;
}

export function createNetworking(inputs: NetworkingInputs): NetworkingOutputs {
  const {
    region,
    lugiaDomain,
    giratinaDomain,
    lugiaService,
    giratinaService,
    apis,
  } = inputs;

  const staticIp = new gcp.compute.GlobalAddress(
    "dislyze-lb-ip",
    {
      name: "dislyze-lb-ip",
    },
    { dependsOn: apis }
  );

  const sslPolicy = new gcp.compute.SSLPolicy(
    "ssl-policy",
    {
      name: "ssl-policy",
      profile: "MODERN",
      minTlsVersion: "TLS_1_2",
    },
    { dependsOn: apis }
  );

  const lugiaCert = new gcp.compute.ManagedSslCertificate(
    "lugia-cert",
    {
      managed: {
        domains: [lugiaDomain],
      },
    },
    { dependsOn: apis }
  );

  const giratinaCert = new gcp.compute.ManagedSslCertificate(
    "giratina-cert",
    {
      managed: {
        domains: [giratinaDomain],
      },
    },
    { dependsOn: apis }
  );

  const lugiaServerlessNeg = new gcp.compute.RegionNetworkEndpointGroup(
    "lugia-serverless-neg",
    {
      region: region,
      networkEndpointType: "SERVERLESS",
      cloudRun: {
        service: lugiaService.name,
      },
    },
    { dependsOn: [lugiaService] }
  );

  const giratinaServerlessNeg = new gcp.compute.RegionNetworkEndpointGroup(
    "giratina-serverless-neg",
    {
      region: region,
      networkEndpointType: "SERVERLESS",
      cloudRun: {
        service: giratinaService.name,
      },
    },
    { dependsOn: [giratinaService] }
  );

  const lugiaBackendService = new gcp.compute.BackendService(
    "lugia-backend-service",
    {
      loadBalancingScheme: "EXTERNAL_MANAGED",
      protocol: "HTTP",
      timeoutSec: 30,
      backends: [
        {
          group: lugiaServerlessNeg.id,
        },
      ],
    },
    { dependsOn: [lugiaServerlessNeg] }
  );

  const giratinaBackendService = new gcp.compute.BackendService(
    "giratina-backend-service",
    {
      loadBalancingScheme: "EXTERNAL_MANAGED",
      protocol: "HTTP",
      timeoutSec: 30,
      backends: [
        {
          group: giratinaServerlessNeg.id,
        },
      ],
    },
    { dependsOn: [giratinaServerlessNeg] }
  );

  const securityHeaders = [
    {
      headerName: "Strict-Transport-Security",
      headerValue: "max-age=31536000; includeSubDomains",
      replace: false,
    },
    {
      headerName: "X-Frame-Options",
      headerValue: "DENY",
      replace: false,
    },
    {
      headerName: "X-Content-Type-Options",
      headerValue: "nosniff",
      replace: false,
    },
    {
      headerName: "Content-Security-Policy",
      headerValue:
        "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
      replace: false,
    },
    {
      headerName: "X-XSS-Protection",
      headerValue: "1; mode=block",
      replace: false,
    },
  ];

  const urlMap = new gcp.compute.URLMap(
    "url-map",
    {
      defaultService: lugiaBackendService.id,
      hostRules: [
        {
          hosts: [lugiaDomain],
          pathMatcher: "lugia",
        },
        {
          hosts: [giratinaDomain],
          pathMatcher: "giratina",
        },
      ],
      pathMatchers: [
        {
          name: "lugia",
          defaultService: lugiaBackendService.id,
          headerAction: {
            responseHeadersToAdds: securityHeaders,
          },
        },
        {
          name: "giratina",
          defaultService: giratinaBackendService.id,
          headerAction: {
            responseHeadersToAdds: securityHeaders,
          },
        },
      ],
    },
    { dependsOn: [lugiaBackendService, giratinaBackendService] }
  );

  const httpsProxy = new gcp.compute.TargetHttpsProxy(
    "https-proxy",
    {
      urlMap: urlMap.id,
      sslCertificates: [lugiaCert.id, giratinaCert.id],
      sslPolicy: sslPolicy.id,
    },
    { dependsOn: [urlMap, lugiaCert, giratinaCert, sslPolicy] }
  );

  new gcp.compute.GlobalForwardingRule(
    "https-forwarding-rule",
    {
      target: httpsProxy.id,
      portRange: "443",
      ipProtocol: "TCP",
      ipAddress: staticIp.address,
      loadBalancingScheme: "EXTERNAL_MANAGED",
    },
    { dependsOn: [httpsProxy, staticIp] }
  );

  const redirectUrlMap = new gcp.compute.URLMap("redirect-url-map", {
    defaultUrlRedirect: {
      httpsRedirect: true,
      stripQuery: false,
    },
  });

  const httpProxy = new gcp.compute.TargetHttpProxy(
    "http-proxy",
    {
      urlMap: redirectUrlMap.id,
    },
    { dependsOn: [redirectUrlMap] }
  );

  new gcp.compute.GlobalForwardingRule(
    "http-forwarding-rule",
    {
      target: httpProxy.id,
      portRange: "80",
      ipProtocol: "TCP",
      ipAddress: staticIp.address,
      loadBalancingScheme: "EXTERNAL_MANAGED",
    },
    { dependsOn: [httpProxy, staticIp] }
  );

  return {
    staticIp,
    loadBalancerIp: staticIp.address,
  };
}
