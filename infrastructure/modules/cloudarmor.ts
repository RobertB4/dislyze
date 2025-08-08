import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface CloudArmorInputs {
  projectId: string | pulumi.Output<string>;
  environment: string;
  apis: gcp.projects.Service[];
}

export interface CloudArmorOutputs {
  securityPolicy: gcp.compute.SecurityPolicy;
}

export function createCloudArmor(inputs: CloudArmorInputs): CloudArmorOutputs {
  const { environment, apis } = inputs;

  // Main security policy with adaptive protection enabled
  const securityPolicy = new gcp.compute.SecurityPolicy(
    "dislyze-armor-policy", 
    {
      name: `dislyze-armor-policy-${environment}`,
      description: "Cloud Armor security policy with WAF rules and adaptive protection",
      
      // Enable adaptive protection for ML-based DDoS detection
      adaptiveProtectionConfig: {
        layer7DdosDefenseConfig: {
          enable: true,
          ruleVisibility: "STANDARD",
        },
      },
      
      // Advanced DDoS protection settings
      advancedOptionsConfig: {
        logLevel: "VERBOSE",
        jsonParsing: "STANDARD",
        jsonCustomConfig: {
          contentTypes: ["application/json", "application/x-www-form-urlencoded"],
        },
        userIpRequestHeaders: ["X-Forwarded-For", "X-Real-IP"],
      },
    },
    { dependsOn: apis }
  );

  // Combined OWASP CRS 3.3 Protection (Priority 100)
  new gcp.compute.SecurityPolicyRule("owasp-protection-rule", {
    securityPolicy: securityPolicy.name,
    priority: 100,
    action: "deny(403)",
    match: {
      expr: {
        expression: `
          evaluatePreconfiguredWaf('sqli-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('xss-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('lfi-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('rfi-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('rce-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('php-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('protocolattack-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('sessionfixation-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('methodenforcement-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('scannerdetection-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('java-v33-stable', {'sensitivity': 1}) ||
          evaluatePreconfiguredWaf('nodejs-v33-stable', {'sensitivity': 1})
        `.replace(/\s+/g, ' ').trim(),
      },
    },
    description: "Block all OWASP CRS 3.3 attacks: SQL injection, XSS, file inclusion, RCE, PHP/Java/NodeJS injection, protocol attacks, session fixation, method enforcement, and scanner detection",
  }, { dependsOn: [securityPolicy] });

  // Rate limiting rule (Priority 1000) - Prevent brute force attacks
  new gcp.compute.SecurityPolicyRule("rate-limiting-rule", {
    securityPolicy: securityPolicy.name,
    priority: 1000,
    action: "rate_based_ban",
    rateLimitOptions: {
      conformAction: "allow",
      exceedAction: "deny(429)",
      exceedRedirectOptions: {
        type: "EXTERNAL_302",
        target: "https://www.example.com/rate-limit-exceeded",
      },
      enforceOnKey: "IP",
      rateLimitThreshold: {
        count: 100, // 100 requests
        intervalSec: 60, // per minute
      },
      banThreshold: {
        count: 1000, // Ban after 1000 requests
        intervalSec: 600, // in 10 minutes
      },
      banDurationSec: 600, // Ban for 10 minutes
    },
    match: {
      versionedExpr: "SRC_IPS_V1",
      config: {
        srcIpRanges: ["*"],
      },
    },
    description: "Rate limiting: 100 req/min, ban after 1000 req/10min",
  }, { dependsOn: [securityPolicy] });

  // Geography-based rule (Priority 1100) - Optional: Restrict access by region
  // Uncomment if you need to restrict access to specific countries
  /*
  new gcp.compute.SecurityPolicyRule("geo-restriction-rule", {
    securityPolicy: securityPolicy.name,
    priority: 1100,
    action: "deny(403)",
    match: {
      expr: {
        expression: "origin.region_code != 'JP' && origin.region_code != 'US'",
      },
    },
    description: "Restrict access to Japan and US only",
  }, { dependsOn: [securityPolicy] });
  */

  // Default allow rule (Priority 2147483647 - must be maximum)
  new gcp.compute.SecurityPolicyRule("default-allow-rule", {
    securityPolicy: securityPolicy.name,
    priority: 2147483647,
    action: "allow",
    match: {
      versionedExpr: "SRC_IPS_V1",
      config: {
        srcIpRanges: ["*"],
      },
    },
    description: "Default allow all traffic",
  }, { dependsOn: [securityPolicy] });

  return {
    securityPolicy,
  };
}