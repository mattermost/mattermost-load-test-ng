// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import "fmt"

const mattermostServiceFile = `
[Unit]
Description=Mattermost
After=network.target

[Service]
Type=simple
ExecStart=/opt/mattermost/bin/mattermost
Restart=always
RestartSec=10
WorkingDirectory=/opt/mattermost
User=ubuntu
Group=ubuntu
LimitNOFILE=49152
Environment=MM_FEATUREFLAGS_POSTPRIORITY=true
Environment=MM_FEATUREFLAGS_WEBSOCKETEVENTSCOPE=true
Environment=MM_FEATUREFLAGS_CHANNELBOOKMARKS=true
Environment=MM_SERVICEENVIRONMENT=%s

[Install]
WantedBy=multi-user.target
`

const prometheusConfig = `
global:
  scrape_interval:     5s
  evaluation_interval: 5s

# A scrape configuration containing exactly one endpoint to scrape:
# Here it's Prometheus itself.
scrape_configs:
  - job_name: prometheus
    static_configs:
        - targets: ['metrics:9090']
  - job_name: node
    static_configs:
        - targets: ['metrics:9100',%s]
  - job_name: mattermost
    static_configs:
        - targets: [%s]
  - job_name: elasticsearch
    static_configs:
        - targets: [%s]
  - job_name: loadtest
    static_configs:
        - targets: [%s]
  - job_name: keycloak
    static_configs:
        - targets: [%s]
  - job_name: redis
    static_configs:
        - targets: [%s]
  - job_name: cloudwatch
    static_configs:
        - targets: [%s]
  - job_name: netpeek
    static_configs:
        - targets: [%s]
`

const metricsHosts = `
127.0.0.1 localhost

# The following lines are desirable for IPv6 capable hosts
::1 ip6-localhost ip6-loopback
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
ff02::3 ip6-allhosts

127.0.0.1 metrics
%s
`

const appHosts = `
127.0.0.1 localhost

# The following lines are desirable for IPv6 capable hosts
::1 ip6-localhost ip6-loopback
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
ff02::3 ip6-allhosts

%s
`

const nginxConfigTmpl = `
user www-data;
worker_processes auto;
worker_rlimit_nofile 100000;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;

events {
  worker_connections 20000;
  use epoll;
}

http {
  map $status $loggable {
    ~^[23] 0;
    default 1;
  }

  log_format json escape=json
	'{'
		'"ts":"$msec",'
		'"time_local":"$time_local",'
		'"remote_addr":"$remote_addr",'
		'"request":"$request",'
		'"request_time":"$request_time",'
		'"status": "$status",'
		'"body_bytes_sent":"$body_bytes_sent",'
		'"upstream_addr":"$upstream_addr",'
		'"upstream_status":"$upstream_status",'
		'"upstream_response_time":"$upstream_response_time",'
		'"upstream_cache_status":"$upstream_cache_status",'
		'"http_user_agent":"$http_user_agent"'
	'}';

  sendfile on;
  tcp_nopush on;
  tcp_nodelay {{.tcpNoDelay}};
  keepalive_timeout 75s;
  keepalive_requests 16384;
  types_hash_max_size 2048;
  include /etc/nginx/mime.types;
  default_type application/octet-stream;
  ssl_prefer_server_ciphers on;
  access_log /var/log/nginx/access.log json if=$loggable;
  error_log /var/log/nginx/error.log;
  gzip on;
  include /etc/nginx/sites-enabled/*;
}
`

const nginxProxyCommonConfig = `
client_max_body_size 50M;
proxy_set_header Host $http_host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Frame-Options SAMEORIGIN;
proxy_buffers 256 16k;
proxy_buffer_size 16k;
client_body_timeout 60s;
send_timeout        300s;
lingering_timeout   30s;
proxy_connect_timeout   30s;
proxy_send_timeout      90s;
proxy_read_timeout      90s;
proxy_http_version 1.1;
proxy_pass http://backend;
`

const nginxCacheCommonConfig = `
proxy_cache mattermost_cache;
proxy_cache_revalidate on;
proxy_cache_min_uses 2;
proxy_cache_use_stale timeout;
proxy_cache_lock on;
`

const nginxSiteConfigTmpl = `
upstream backend {
{{.backends}}
  keepalive 256;
}

proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=mattermost_cache:{{.cacheObjects}} max_size={{.cacheSize}} inactive=60m use_temp_path=off;

server {
  listen 80 reuseport;
  server_name _;

  location ~ /api/v[0-9]+/(users/)?websocket$ {
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    include /etc/nginx/snippets/proxy.conf;
  }

  location ~ /api/v[0-9]+/users/[a-z0-9]+/image$ {
    proxy_set_header Connection "";
    include /etc/nginx/snippets/proxy.conf;
    include /etc/nginx/snippets/cache.conf;
    proxy_ignore_headers Cache-Control Expires;
    proxy_cache_valid 200 24h;
  }

  location / {
    proxy_set_header Connection "";
    include /etc/nginx/snippets/proxy.conf;
    include /etc/nginx/snippets/cache.conf;
  }
}
`

const limitsConfig = `
* soft nofile 100000
* hard nofile 100000
* soft nproc 8192
* hard nproc 8192
`

const clientSysctlConfig = `
# Extending default port range to handle lots of concurrent connections.
net.ipv4.ip_local_port_range = 1025 65000

# Lowering the timeout to faster recycle connections in the FIN-WAIT-2 state.
net.ipv4.tcp_fin_timeout = 30

# Reuse TIME-WAIT sockets for new outgoing connections.
net.ipv4.tcp_tw_reuse = 1

# TCP buffer sizes are tuned for 10Gbit/s bandwidth and 0.5ms RTT (as measured intra EC2 cluster).
# This gives a BDP (bandwidth-delay-product) of 625000 bytes.
net.ipv4.tcp_rmem = 4096 156250 625000
net.ipv4.tcp_wmem = 4096 156250 625000
net.core.rmem_max = 312500
net.core.wmem_max = 312500
net.core.rmem_default = 312500
net.core.wmem_default = 312500
net.ipv4.tcp_mem = 1638400 1638400 1638400
`

const serverSysctlConfig = `
# Extending default port range to handle lots of concurrent connections.
net.ipv4.ip_local_port_range = 1025 65000

# Lowering the timeout to faster recycle connections in the FIN-WAIT-2 state.
net.ipv4.tcp_fin_timeout = 30

# Reuse TIME-WAIT sockets for new outgoing connections.
net.ipv4.tcp_tw_reuse = 1

# Bumping the limit of a listen() backlog.
# This is maximum number of established sockets (with an ACK)
# waiting to be accepted by the listening process.
net.core.somaxconn = 4096

# Increasing the maximum number of connection requests which have
# not received an acknowledgment from the client.
# This is helpful to handle sudden bursts of new incoming connections.
net.ipv4.tcp_max_syn_backlog = 8192

# This is tuned to be 2% of the available memory.
vm.min_free_kbytes = 167772

# Disabling slow start helps increasing overall throughput
# and performance of persistent single connections.
net.ipv4.tcp_slow_start_after_idle = 0

# These show a good performance improvement over defaults.
# More info at https://blog.cloudflare.com/http-2-prioritization-with-nginx/
net.ipv4.tcp_congestion_control = bbr
net.core.default_qdisc = fq
net.ipv4.tcp_notsent_lowat = 16384

# TCP buffer sizes are tuned for 10Gbit/s bandwidth and 0.5ms RTT (as measured intra EC2 cluster).
# This gives a BDP (bandwidth-delay-product) of 625000 bytes.
# The maximum socket buffer size for kernel autotuning is set to be 4x the BDP (2500000).
# The default socket buffer size is set to 1/4 BDP (156250).
net.ipv4.tcp_rmem = 4096 156250 2500000
net.ipv4.tcp_wmem = 4096 156250 2500000

# Bumping the theoretical maximum buffer size for receiving TCP sockets not making use of autotuning (i.e. using SO_RCVBUF).
net.core.rmem_max = 2500000
# Bumping the theoretical maximum buffer size for sending TCP sockets not making use of autotuning (i.e. using SO_SNDBUF).
net.core.wmem_max = 2500000

# Bumping the theoretical maximum buffer size of receiving UDP sockets.
net.core.rmem_max = 16777216

# Setting the theoretical maximum buffer size of sending UDP sockets.
net.core.wmem_max = 16777216
`

const baseAPIServerCmd = `/home/ubuntu/mattermost-load-test-ng/bin/ltapi`

const apiServiceFile = `
[Unit]
Description=Mattermost load-test API Server
After=network.target

[Service]
Type=simple
Environment="GOGC=50"
Environment="BLOCK_PROFILE_RATE={{ printf "%d" .blockProfileRate}}"
ExecStart={{ printf "%s" .execStart}}
Restart=always
RestartSec=1
WorkingDirectory=/home/ubuntu/mattermost-load-test-ng
User=ubuntu
Group=ubuntu
LimitNOFILE=262144

[Install]
WantedBy=multi-user.target
`

const esExporterServiceFile = `
[Unit]
Description=Elasticsearch prometheus exporter
After=network.target

[Service]
Type=simple
ExecStart=/opt/elasticsearch_exporter/elasticsearch_exporter --es.uri="%s"
Restart=always
RestartSec=10
WorkingDirectory=/opt/elasticsearch_exporter
User=ubuntu
Group=ubuntu

[Install]
WantedBy=multi-user.target
`

const redisExporterServiceFile = `
[Unit]
Description=Redis prometheus exporter
After=network.target

[Service]
Type=simple
ExecStart=/opt/redis_exporter/redis_exporter --redis.addr="%s"
Restart=always
RestartSec=10
WorkingDirectory=/opt/redis_exporter
User=ubuntu
Group=ubuntu

[Install]
WantedBy=multi-user.target
`

const yaceConfigFile = `
apiVersion: v1alpha1
discovery:
  exportedTagsOnMetrics:
    AWS/RDS:
      - Name
      - ClusterName
    AWS/ES:
      - Name
      - ClusterName
    AWS/EC2:
      - Name
      - ClusterName
  jobs:
    - type: AWS/RDS
      regions:
        - {{.AWSRegion}}
      period: {{.Period}}
      length: {{.Length}}
      delay: {{.Delay}}
      addCloudwatchTimestamp: true
      searchTags:
        - key: ClusterName
          value: {{.ClusterName}}
      metrics:
        - name: CPUUtilization
          statistics: [Average]
        - name: DatabaseConnections
          statistics: [Sum]
        - name: FreeableMemory
          statistics: [Average]
        - name: FreeStorageSpace
          statistics: [Average]
        - name: ReadThroughput
          statistics: [Average]
        - name: WriteThroughput
          statistics: [Average]
        - name: ReadLatency
          statistics: [Maximum]
        - name: WriteLatency
          statistics: [Maximum]
        - name: ReadIOPS
          statistics: [Average]
        - name: WriteIOPS
          statistics: [Average]
    - type: AWS/ES
      regions:
        - {{.AWSRegion}}
      period: {{.Period}}
      length: {{.Length}}
      delay: {{.Delay}}
      addCloudwatchTimestamp: true
      searchTags:
        - key: ClusterName
          value: {{.ClusterName}}
      metrics:
        - name: CPUUtilization
          statistics: [Average]
        - name: FreeStorageSpace
          statistics: [Sum]
        - name: ClusterStatus.green
          statistics: [Maximum]
        - name: ClusterStatus.yellow
          statistics: [Maximum]
        - name: ClusterStatus.red
          statistics: [Maximum]
        - name: Shards.active
          statistics: [Sum]
        - name: Shards.unassigned
          statistics: [Sum]
        - name: Shards.delayedUnassigned
          statistics: [Sum]
        - name: Shards.activePrimary
          statistics: [Sum]
        - name: Shards.initializing
          statistics: [Sum]
        - name: Shards.initializing
          statistics: [Sum]
        - name: Shards.relocating
          statistics: [Sum]
        - name: Nodes
          statistics: [Maximum]
        - name: SearchableDocuments
          statistics: [Maximum]
        - name: DeletedDocuments
          statistics: [Maximum]
    - type: AWS/EC2
      regions:
        - {{.AWSRegion}}
      period: {{.Period}}
      length: {{.Length}}
      delay: {{.Delay}}
      addCloudwatchTimestamp: true
      nilToZero: true
      searchTags:
        - key: ClusterName
          value: {{.ClusterName}}
      metrics:
        - name: StatusCheckFailed
          statistics: [Sum]
        - name: StatusCheckFailed_Instance
          statistics: [Sum]
        - name: StatusCheckFailed_System
          statistics: [Sum]
`

const yaceServiceFile = `
[Unit]
Description=Cloudwatch prometheus exporter - YACE
After=network.target

[Service]
Type=simple
ExecStart=/opt/yace/yace -listen-address :{{.Port}} -config.file /opt/yace/conf.yml -scraping-interval {{.ScrapingInterval}}
Restart=always
RestartSec=10
WorkingDirectory=/opt/yace
User=ubuntu
Group=ubuntu

[Install]
WantedBy=multi-user.target
`

const jobServerServiceFile = `
[Unit]
Description=Mattermost Job Server
After=network.target

[Service]
Type=simple
ExecStart=/opt/mattermost/bin/mattermost jobserver
Restart=always
RestartSec=10
WorkingDirectory=/opt/mattermost
User=ubuntu
Group=ubuntu
LimitNOFILE=49152
Environment=MM_SERVICEENVIRONMENT=%s

[Install]
WantedBy=multi-user.target
`

const grafanaConfigFile = `
[auth]
disable_login_form = false

[auth.anonymous]
enabled = true
org_role = Editor

[dashboards]
default_home_dashboard_path = /var/lib/grafana/dashboards/dashboard.json
`

const keycloakServiceFileContents = `
[Unit]
Description=Keycloak
After=network.target

[Service]
User=ubuntu
Group=ubuntu
EnvironmentFile=/etc/systemd/system/keycloak.env
ExecStart=/opt/keycloak/keycloak-{{ .KeycloakVersion }}/bin/kc.sh {{ .Command }}

[Install]
WantedBy=multi-user.target
`

const keycloakEnvFileContents = `KC_HEALTH_ENABLED=true
KEYCLOAK_ADMIN={{ .KeycloakAdminUser }}
KEYCLOAK_ADMIN_PASSWORD={{ .KeycloakAdminPassword }}
JAVA_OPTS=-Xms1024m -Xmx2048m
KC_LOG_FILE={{ .KeycloakLogFilePath }}
KC_LOG_FILE_OUTPUT=json
KC_DB_POOL_MIN_SIZE=20
KC_DB_POOL_INITIAL_SIZE=20
KC_DB_POOL_MAX_SIZE=200
KC_DB=postgres
KC_DB_URL=jdbc:psql://localhost:5433/keycloak
KC_DB_PASSWORD=mmpass
KC_DB_USERNAME=keycloak
KC_DATABASE=keycloak`

const prometheusNodeExporterConfig = `
ARGS="--collector.ethtool"
`

const netpeekServiceFile = `
[Unit]
Description=netpeek
After=network.target

[Service]
Type=simple
ExecStart=/bin/sh -c '/usr/local/bin/netpeek -iface "$(ip route show to default | awk \'{print $5}\')" -port %d'
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
`

const otelcolOperatorAppAgent = `
      - type: json_parser
        timestamp:
          parse_from: attributes.timestamp
          layout: '%Y-%m-%d %H:%M:%S.%L Z'
        severity:
          parse_from: attributes.level`

const otelcolOperatorProxyError = `
      - type: regex_parser
        regex: '^(?P<time>\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}) \[(?P<sev>[a-z]*)\] (?P<msg>.*)$'
        timestamp:
          parse_from: attributes.time
          layout: '%Y/%m/%d %H:%M:%S'
        severity:
          parse_from: attributes.sev`

const otelcolOperatorProxyAccess = `
      - type: json_parser
        timestamp:
          layout: 's.ms'
          layout_type: epoch
          parse_from: attributes.ts`

const otelcolConfigTmpl = `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
{{range .Receivers}}
  {{.Name}}:
    include: [ {{.IncludeFiles}} ]
    resource:
      service.name: "{{.ServiceName}}"
      service.instance.id: "{{.ServiceInstanceId}}"
    operators:{{.Operator}}{{end}}

exporters:
  otlphttp/logs:
    endpoint: "http://{{.MetricsIP}}:3100/otlp"
    tls:
      insecure: true
  debug:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200

service:
  pipelines:
    logs:
      receivers: [{{range .Receivers}}{{.Name}},{{end}}]
      exporters: [otlphttp/logs,debug]
`

type otelcolReceiver struct {
	Name              string
	IncludeFiles      string
	ServiceName       string
	ServiceInstanceId string
	Operator          string
}

func renderAgentOtelcolConfig(instanceName string, metricsIP string) (string, error) {
	agentReceiver := otelcolReceiver{
		Name:              "filelog/agent",
		IncludeFiles:      "/home/ubuntu/mattermost-load-test-ng/ltagent.log",
		ServiceName:       "agent",
		ServiceInstanceId: instanceName,
		Operator:          otelcolOperatorAppAgent,
	}

	coordinatorReceiver := otelcolReceiver{
		Name:              "filelog/coordinator",
		IncludeFiles:      "/home/ubuntu/mattermost-load-test-ng/ltcoordinator.log",
		ServiceName:       "coordinator",
		ServiceInstanceId: instanceName,
		Operator:          otelcolOperatorAppAgent,
	}

	otelcolConfig, err := fillConfigTemplate(otelcolConfigTmpl, map[string]any{
		"Receivers": []otelcolReceiver{agentReceiver, coordinatorReceiver},
		"MetricsIP": metricsIP,
	})
	if err != nil {
		return "", fmt.Errorf("unable to render otelcol config template")
	}

	return otelcolConfig, err
}

func renderProxyOtelcolConfig(instanceName string, metricsIP string) (string, error) {
	proxyErrorReceiver := otelcolReceiver{
		Name:              "filelog/proxy_error",
		IncludeFiles:      "/var/log/nginx/error.log",
		ServiceName:       "proxy",
		ServiceInstanceId: instanceName,
		Operator:          otelcolOperatorProxyError,
	}

	proxyAccessReceiver := otelcolReceiver{
		Name:              "filelog/proxy_access",
		IncludeFiles:      "/var/log/nginx/access.log",
		ServiceName:       "proxy",
		ServiceInstanceId: instanceName,
		Operator:          otelcolOperatorProxyAccess,
	}

	otelcolConfig, err := fillConfigTemplate(otelcolConfigTmpl, map[string]any{
		"Receivers": []otelcolReceiver{proxyErrorReceiver, proxyAccessReceiver},
		"MetricsIP": metricsIP,
	})
	if err != nil {
		return "", fmt.Errorf("unable to render otelcol config template")
	}

	return otelcolConfig, nil
}

func renderAppOtelcolConfig(instanceName string, metricsIP string) (string, error) {
	appReceiver := otelcolReceiver{
		Name:              "filelog/app",
		IncludeFiles:      "/opt/mattermost/logs/mattermost.log",
		ServiceName:       "app",
		ServiceInstanceId: instanceName,
		Operator:          otelcolOperatorAppAgent,
	}

	otelcolConfig, err := fillConfigTemplate(otelcolConfigTmpl, map[string]any{
		"Receivers": []otelcolReceiver{appReceiver},
		"MetricsIP": metricsIP,
	})
	if err != nil {
		return "", fmt.Errorf("unable to render otelcol config template: %w", err)
	}

	return otelcolConfig, nil
}
