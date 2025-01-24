package terraform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	expectedAgentConf = `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

  filelog/agent:
    include: [ /home/ubuntu/mattermost-load-test-ng/ltagent.log ]
    resource:
      service.name: "agent"
      service.instance.id: "instance-name"
    operators:
      - type: json_parser
        timestamp:
          parse_from: attributes.timestamp
          layout: '%Y-%m-%d %H:%M:%S.%L Z'
        severity:
          parse_from: attributes.level
  filelog/coordinator:
    include: [ /home/ubuntu/mattermost-load-test-ng/ltcoordinator.log ]
    resource:
      service.name: "coordinator"
      service.instance.id: "instance-name"
    operators:
      - type: json_parser
        timestamp:
          parse_from: attributes.timestamp
          layout: '%Y-%m-%d %H:%M:%S.%L Z'
        severity:
          parse_from: attributes.level

exporters:
  otlphttp/logs:
    endpoint: "http://127.0.0.1:3100/otlp"
    tls:
      insecure: true

service:
  pipelines:
    logs:
      receivers: [filelog/agent,filelog/coordinator,]
      exporters: [otlphttp/logs]
`
	expectedAppConfg = `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

  filelog/app:
    include: [ /opt/mattermost/logs/mattermost.log ]
    resource:
      service.name: "app"
      service.instance.id: "instance-name"
    operators:
      - type: json_parser
        timestamp:
          parse_from: attributes.timestamp
          layout: '%Y-%m-%d %H:%M:%S.%L Z'
        severity:
          parse_from: attributes.level

exporters:
  otlphttp/logs:
    endpoint: "http://127.0.0.1:3100/otlp"
    tls:
      insecure: true
  debug:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200

service:
  pipelines:
    logs:
      receivers: [filelog/app,]
      exporters: [otlphttp/logs,debug]
`

	expectedProxyConf = `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

  filelog/proxy_error:
    include: [ /var/log/nginx/error.log ]
    resource:
      service.name: "proxy"
      service.instance.id: "instance-name"
    operators:
      - type: regex_parser
        regex: '^(?P<time>\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}) \[(?P<sev>[a-z]*)\] (?P<msg>.*)$'
        timestamp:
          parse_from: attributes.time
          layout: '%Y/%m/%d %H:%M:%S'
        severity:
          parse_from: attributes.sev
  filelog/proxy_access:
    include: [ /var/log/nginx/access.log ]
    resource:
      service.name: "proxy"
      service.instance.id: "instance-name"
    operators:
      - type: json_parser
        timestamp:
          layout: 's.ms'
          layout_type: epoch
          parse_from: attributes.ts

exporters:
  otlphttp/logs:
    endpoint: "http://127.0.0.1:3100/otlp"
    tls:
      insecure: true
  debug:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200

service:
  pipelines:
    logs:
      receivers: [filelog/proxy_error,filelog/proxy_access,]
      exporters: [otlphttp/logs,debug]
`
)

func TestRenderAgentOtelcolConfig(t *testing.T) {
	instanceName := "instance-name"
	metricsIP := "127.0.0.1"
	cfg, err := renderAgentOtelcolConfig(instanceName, metricsIP)
	require.NoError(t, err)

	require.Equal(t, expectedAgentConf, cfg)
}

func TestRenderAppOtelcolConfig(t *testing.T) {
	instanceName := "instance-name"
	metricsIP := "127.0.0.1"
	cfg, err := renderAppOtelcolConfig(instanceName, metricsIP)
	require.NoError(t, err)

	require.Equal(t, expectedAppConfg, cfg)
}

func TestRenderProxyOtelcolConfig(t *testing.T) {
	instanceName := "instance-name"
	metricsIP := "127.0.0.1"
	cfg, err := renderProxyOtelcolConfig(instanceName, metricsIP)
	require.NoError(t, err)

	require.Equal(t, expectedProxyConf, cfg)
}
