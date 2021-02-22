// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

const serviceFile = `
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

const nginxConfig = `
user www-data;
worker_processes auto;
worker_rlimit_nofile 65536;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;

events {
	worker_connections 16384;
	use epoll;
}


http {
  map $status $loggable {
    ~^[23] 0;
    default 1;
  }

	sendfile on;
	tcp_nopush on;
	tcp_nodelay on;
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
     client_max_body_size 50M;
     proxy_set_header Host $http_host;
     proxy_set_header X-Real-IP $remote_addr;
     proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
     proxy_set_header X-Forwarded-Proto $scheme;
     proxy_set_header X-Frame-Options SAMEORIGIN;
     proxy_buffers 256 16k;
     proxy_buffer_size 16k;
     client_body_timeout 60;
     send_timeout        300;
     lingering_timeout   5;
     proxy_connect_timeout   30s;
     proxy_send_timeout      90s;
     proxy_read_timeout      90s;
     proxy_http_version 1.1;
     proxy_pass http://backend;
   }

   location / {
     client_max_body_size 50M;
     proxy_set_header Connection "";
     proxy_set_header Host $http_host;
     proxy_set_header X-Real-IP $remote_addr;
     proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
     proxy_set_header X-Forwarded-Proto $scheme;
     proxy_set_header X-Frame-Options SAMEORIGIN;
     proxy_buffers 256 16k;
     proxy_buffer_size 16k;
     proxy_connect_timeout   30s;
     proxy_read_timeout      90s;
     proxy_send_timeout      90s;
     proxy_cache mattermost_cache;
     proxy_cache_revalidate on;
     proxy_cache_min_uses 2;
     proxy_cache_use_stale timeout;
     proxy_cache_lock on;
     proxy_http_version 1.1;
     proxy_pass http://backend;
   }
}
`

const limitsConfig = `
* soft nofile 65536
* hard nofile 65536
* soft nproc 8192
* hard nproc 8192
`

const clientSysctlConfig = `
net.ipv4.ip_local_port_range = 1025 65000
net.ipv4.tcp_fin_timeout = 30
`

const serverSysctlConfig = `
net.ipv4.ip_local_port_range = 1025 65000
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 8192
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
LimitNOFILE=65536

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
