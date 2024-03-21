// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

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
`

type PyroscopeConfig struct {
	LogLevel        string         `yaml:"log-level"`
	NoSelfProfiling bool           `yaml:"no-self-profiling"`
	ScrapeConfigs   []ScrapeConfig `yaml:"scrape-configs"`
}

type ScrapeConfig struct {
	JobName         string         `yaml:"job-name"`
	Scheme          string         `yaml:"scheme"`
	ScrapeInterval  string         `yaml:"scrape-interval"`
	EnabledProfiles []string       `yaml:"enabled-profiles,flow"`
	StaticConfigs   []StaticConfig `yaml:"static-configs,omitempty"`
}

type StaticConfig struct {
	Application string   `yaml:"application"`
	SpyName     string   `yaml:"spy-name"`
	Targets     []string `yaml:"targets,flow"`
}

func NewPyroscopeConfig(mmTargets, ltTargets []string) *PyroscopeConfig {
	var staticConfigs []StaticConfig

	if len(mmTargets) > 0 {
		staticConfigs = append(staticConfigs, StaticConfig{
			Application: "mattermost",
			SpyName:     "gospy",
			Targets:     mmTargets,
		})
	}

	if len(ltTargets) > 0 {
		staticConfigs = append(staticConfigs, StaticConfig{
			Application: "agents",
			SpyName:     "gospy",
			Targets:     ltTargets,
		})
	}

	return &PyroscopeConfig{
		LogLevel:        "debug",
		NoSelfProfiling: true,
		ScrapeConfigs: []ScrapeConfig{
			{
				JobName:         "pryoscope",
				Scheme:          "http",
				ScrapeInterval:  "60s",
				EnabledProfiles: []string{"cpu", "mem", "goroutines"},
				StaticConfigs:   staticConfigs,
			},
		},
	}
}

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

  sendfile on;
  tcp_nopush on;
  tcp_nodelay {{.tcpNoDelay}};
  keepalive_timeout 75s;
  keepalive_requests 16384;
  types_hash_max_size 2048;
  include /etc/nginx/mime.types;
  default_type application/octet-stream;
  ssl_prefer_server_ciphers on;
  access_log /var/log/nginx/access.log combined if=$loggable;
  error_log /var/log/nginx/error.log;
  gzip on;
  include /etc/nginx/conf.d/*.conf;
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
lingering_timeout   5s;
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

const nginxSiteConfig = `
upstream backend {
%s
  keepalive 256;
}

proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=mattermost_cache:10m max_size=3g inactive=120m use_temp_path=off;

server {
  listen 80;
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
net.ipv4.tcp_rmem = 4096 156250 625000
net.ipv4.tcp_wmem = 4096 156250 625000
net.core.rmem_max = 312500
net.core.wmem_max = 312500
net.core.rmem_default = 312500
net.core.wmem_default = 312500
net.ipv4.tcp_mem = 1638400 1638400 1638400
`

const baseAPIServerCmd = `/home/ubuntu/mattermost-load-test-ng/bin/ltapi`

const apiServiceFile = `
[Unit]
Description=Mattermost load-test API Server
After=network.target

[Service]
Type=simple
Environment="GOGC=50"
ExecStart={{ printf "%s" .}}
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
