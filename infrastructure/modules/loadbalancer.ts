import * as gcp from "@pulumi/gcp";
import { GlobalForwardingRule } from "@pulumi/gcp/compute";
import * as pulumi from "@pulumi/pulumi";

export interface LoadBalancerInputs {
  region: string | pulumi.Output<string>;
  lugiaDomain: string;
  giratinaDomain: string;
  lugiaService: gcp.cloudrun.Service;
  giratinaService: gcp.cloudrun.Service;
  apis: gcp.projects.Service[];
  securityPolicy: gcp.compute.SecurityPolicy;
}

export interface LoadBalancerOutputs {
  staticIp: gcp.compute.GlobalAddress;
  loadBalancerIp: pulumi.Output<string>;
  lugiaCert: gcp.compute.ManagedSslCertificate;
  giratinaCert: gcp.compute.ManagedSslCertificate;
  httpForwardingRule: GlobalForwardingRule;
  httpsForwardingRule: GlobalForwardingRule;
}

export function createLoadBalancer(
  inputs: LoadBalancerInputs
): LoadBalancerOutputs {
  const {
    region,
    lugiaDomain,
    giratinaDomain,
    lugiaService,
    giratinaService,
    apis,
    securityPolicy,
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
      backends: [
        {
          group: lugiaServerlessNeg.id,
        },
      ],
      securityPolicy: securityPolicy.id,
    },
    { dependsOn: [lugiaServerlessNeg, securityPolicy] }
  );

  const giratinaBackendService = new gcp.compute.BackendService(
    "giratina-backend-service",
    {
      loadBalancingScheme: "EXTERNAL_MANAGED",
      protocol: "HTTP",
      backends: [
        {
          group: giratinaServerlessNeg.id,
        },
      ],
      securityPolicy: securityPolicy.id,
    },
    { dependsOn: [giratinaServerlessNeg, securityPolicy] }
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

  const redirectUrlMap = new gcp.compute.URLMap(
    "redirect-url-map",
    {
      defaultUrlRedirect: {
        httpsRedirect: true,
        stripQuery: false,
        redirectResponseCode: "MOVED_PERMANENTLY_DEFAULT",
      },
    },
    { dependsOn: apis }
  );

  const httpProxy = new gcp.compute.TargetHttpProxy(
    "http-proxy",
    {
      urlMap: redirectUrlMap.id,
    },
    { dependsOn: [redirectUrlMap] }
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

  const httpForwardingRule = new gcp.compute.GlobalForwardingRule(
    "http-forwarding-rule",
    {
      target: httpProxy.id,
      portRange: "80",
      ipAddress: staticIp.address,
      loadBalancingScheme: "EXTERNAL_MANAGED",
    },
    { dependsOn: [httpProxy, staticIp] }
  );

  const httpsForwardingRule = new gcp.compute.GlobalForwardingRule(
    "https-forwarding-rule",
    {
      target: httpsProxy.id,
      portRange: "443",
      ipAddress: staticIp.address,
      loadBalancingScheme: "EXTERNAL_MANAGED",
    },
    { dependsOn: [httpsProxy, staticIp] }
  );

  return {
    staticIp,
    loadBalancerIp: staticIp.address,
    lugiaCert,
    giratinaCert,
    httpForwardingRule,
    httpsForwardingRule,
  };
}
