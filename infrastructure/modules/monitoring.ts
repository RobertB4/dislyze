import * as gcp from "@pulumi/gcp";
import * as pulumi from "@pulumi/pulumi";

export interface MonitoringInputs {
  projectId: string | pulumi.Output<string>;
  region: string | pulumi.Output<string>;
  environment: string;
  alertEmail: string;
  lugiaDomain: string;
  giratinaDomain: string;
  lugiaService: gcp.cloudrun.Service;
  giratinaService: gcp.cloudrun.Service;
  dbInstance: gcp.sql.DatabaseInstance;
  apis: gcp.projects.Service[];
}

export interface MonitoringOutputs {
  notificationChannel?: gcp.monitoring.NotificationChannel;
  uptimeChecks?: gcp.monitoring.UptimeCheckConfig[];
  alertPolicies?: gcp.monitoring.AlertPolicy[];
}

export function createMonitoring(inputs: MonitoringInputs): MonitoringOutputs {
  const {
    projectId,
    environment,
    alertEmail,
    lugiaDomain,
    giratinaDomain,
    lugiaService,
    giratinaService,
    dbInstance,
    apis,
  } = inputs;

  // Only create monitoring resources for production environment
  if (environment !== "production") {
    return {};
  }

  const notificationChannel = new gcp.monitoring.NotificationChannel(
    "email-alerts",
    {
      displayName: "Email Alerts",
      type: "email",
      labels: {
        email_address: alertEmail,
      },
      enabled: true,
    },
    { dependsOn: apis }
  );

  const lugiaUptimeCheck = new gcp.monitoring.UptimeCheckConfig(
    "lugia-uptime-check",
    {
      displayName: "Lugia Service Uptime",
      monitoredResource: {
        type: "uptime_url",
        labels: {
          project_id: projectId,
          host: lugiaDomain,
        },
      },
      httpCheck: {
        path: "/api/health",
        port: 443,
        useSsl: true,
        validateSsl: true,
      },
      timeout: "10s",
      period: "300s", // Check every 5 minutes
    },
    { dependsOn: apis }
  );

  const giratinaUptimeCheck = new gcp.monitoring.UptimeCheckConfig(
    "giratina-uptime-check",
    {
      displayName: "Giratina Service Uptime",
      monitoredResource: {
        type: "uptime_url",
        labels: {
          project_id: projectId,
          host: giratinaDomain,
        },
      },
      httpCheck: {
        path: "/api/health",
        port: 443,
        useSsl: true,
        validateSsl: true,
      },
      timeout: "10s",
      period: "300s", // Check every 5 minutes
    },
    { dependsOn: apis }
  );

  const uptimeAlertPolicy = new gcp.monitoring.AlertPolicy(
    "uptime-alert-policy",
    {
      displayName: "Service Uptime Alert",
      combiner: "OR",
      conditions: [
        {
          displayName: "Uptime check failed",
          conditionThreshold: {
            filter: `resource.type="uptime_url" AND metric.type="monitoring.googleapis.com/uptime_check/check_passed"`,
            comparison: "COMPARISON_LT",
            thresholdValue: 1,
            duration: "120s", // Alert after 2 minutes of downtime
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_FRACTION_TRUE",
                crossSeriesReducer: "REDUCE_MEAN",
                groupByFields: ["resource.label.host"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s", // Auto-close after 30 minutes if resolved
      },
    },
    { dependsOn: [notificationChannel, lugiaUptimeCheck, giratinaUptimeCheck] }
  );

  const cloudRunCpuAlert = new gcp.monitoring.AlertPolicy(
    "cloudrun-cpu-alert",
    {
      displayName: "Cloud Run High CPU Usage",
      combiner: "OR",
      conditions: [
        {
          displayName: "CPU usage > 90%",
          conditionThreshold: {
            filter: `resource.type="cloud_run_revision" AND metric.type="run.googleapis.com/container/cpu/utilizations"`,
            comparison: "COMPARISON_GT",
            thresholdValue: 0.9,
            duration: "300s", // Sustained for 5 minutes
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_PERCENTILE_95",
                crossSeriesReducer: "REDUCE_MEAN",
                groupByFields: ["resource.label.service_name"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s",
      },
    },
    { dependsOn: [notificationChannel, lugiaService, giratinaService] }
  );

  const cloudRunMemoryAlert = new gcp.monitoring.AlertPolicy(
    "cloudrun-memory-alert",
    {
      displayName: "Cloud Run High Memory Usage",
      combiner: "OR",
      conditions: [
        {
          displayName: "Memory usage > 85%",
          conditionThreshold: {
            filter: `resource.type="cloud_run_revision" AND metric.type="run.googleapis.com/container/memory/utilizations"`,
            comparison: "COMPARISON_GT",
            thresholdValue: 0.85,
            duration: "300s", // Sustained for 5 minutes
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_PERCENTILE_95",
                crossSeriesReducer: "REDUCE_MEAN",
                groupByFields: ["resource.label.service_name"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s",
      },
    },
    { dependsOn: [notificationChannel, lugiaService, giratinaService] }
  );

  const cloudRunLatencyAlert = new gcp.monitoring.AlertPolicy(
    "cloudrun-latency-alert",
    {
      displayName: "Cloud Run High Request Latency",
      combiner: "OR",
      conditions: [
        {
          displayName: "Request latency > 2000ms (95th percentile)",
          conditionThreshold: {
            filter: `resource.type="cloud_run_revision" AND metric.type="run.googleapis.com/request_latencies"`,
            comparison: "COMPARISON_GT",
            thresholdValue: 2000,
            duration: "300s", // Sustained for 5 minutes
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_DELTA",
                crossSeriesReducer: "REDUCE_PERCENTILE_95",
                groupByFields: ["resource.label.service_name"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s",
      },
    },
    { dependsOn: [notificationChannel, lugiaService, giratinaService] }
  );

  const cloudSqlCpuAlert = new gcp.monitoring.AlertPolicy(
    "cloudsql-cpu-alert",
    {
      displayName: "Cloud SQL High CPU Usage",
      combiner: "OR",
      conditions: [
        {
          displayName: "Database CPU usage > 85%",
          conditionThreshold: {
            filter: `resource.type="cloudsql_database" AND metric.type="cloudsql.googleapis.com/database/cpu/utilization"`,
            comparison: "COMPARISON_GT",
            thresholdValue: 0.85,
            duration: "600s", // Sustained for 10 minutes
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_MEAN",
                crossSeriesReducer: "REDUCE_MEAN",
                groupByFields: ["resource.label.database_id"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s",
      },
    },
    { dependsOn: [notificationChannel, dbInstance] }
  );

  const cloudSqlMemoryAlert = new gcp.monitoring.AlertPolicy(
    "cloudsql-memory-alert",
    {
      displayName: "Cloud SQL High Memory Usage",
      combiner: "OR",
      conditions: [
        {
          displayName: "Database memory usage > 90%",
          conditionThreshold: {
            filter: `resource.type="cloudsql_database" AND metric.type="cloudsql.googleapis.com/database/memory/utilization"`,
            comparison: "COMPARISON_GT",
            thresholdValue: 0.9,
            duration: "600s", // Sustained for 10 minutes
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_MEAN",
                crossSeriesReducer: "REDUCE_MEAN",
                groupByFields: ["resource.label.database_id"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s",
      },
    },
    { dependsOn: [notificationChannel, dbInstance] }
  );

  const cloudSqlDiskAlert = new gcp.monitoring.AlertPolicy(
    "cloudsql-disk-alert",
    {
      displayName: "Cloud SQL High Disk Usage",
      combiner: "OR",
      conditions: [
        {
          displayName: "Database disk usage > 85%",
          conditionThreshold: {
            filter: `resource.type="cloudsql_database" AND metric.type="cloudsql.googleapis.com/database/disk/utilization"`,
            comparison: "COMPARISON_GT",
            thresholdValue: 0.85,
            duration: "300s",
            aggregations: [
              {
                alignmentPeriod: "60s",
                perSeriesAligner: "ALIGN_MEAN",
                crossSeriesReducer: "REDUCE_MEAN",
                groupByFields: ["resource.label.database_id"],
              },
            ],
          },
        },
      ],
      notificationChannels: [notificationChannel.name],
      alertStrategy: {
        autoClose: "1800s",
      },
    },
    { dependsOn: [notificationChannel, dbInstance] }
  );

  const uptimeChecks = [lugiaUptimeCheck, giratinaUptimeCheck];
  const alertPolicies = [
    uptimeAlertPolicy,
    cloudRunCpuAlert,
    cloudRunMemoryAlert,
    cloudRunLatencyAlert,
    cloudSqlCpuAlert,
    cloudSqlMemoryAlert,
    cloudSqlDiskAlert,
  ];

  return {
    notificationChannel,
    uptimeChecks,
    alertPolicies,
  };
}
