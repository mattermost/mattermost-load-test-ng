variable "cluster_name" {
}

variable "cluster_vpc_id" {
}

variable "cluster_subnet_ids" {
  type = object({
    app = list(string)
    job = list(string)
    proxy = list(string)
    agent = list(string)
    elasticsearch = list(string)
    metrics = list(string)
    keycloak = list(string)
    database = list(string)
    redis = list(string)
  })
}

variable "connection_type" {
}

variable "app_attach_iam_profile" {
}

variable "app_instance_count" {
}

variable "app_instance_type" {
}

# Agent variables

variable "agent_instance_count" {
}

variable "agent_instance_type" {
}

variable "agent_allocate_public_ip_address" {
}

# Elasticsearch variables

variable "es_instance_count" {
}

variable "es_instance_type" {
}

variable "es_version" {
}

variable "es_create_role" {
}

variable "es_snapshot_repository" {
}

variable "es_zone_awareness_enabled" {
}

variable "es_zone_awarness_availability_zone_count" {
}

variable "es_enable_cloudwatch_logs" {
}

# Proxy server

variable "proxy_instance_count" {
  type = number
}

# Redis variables

variable "redis_enabled" {
}

variable "redis_node_type" {
}

variable "redis_param_group_name" {
}

variable "redis_engine_version" {
}

variable "proxy_instance_type" {
}

variable "proxy_allocate_public_ip_address" {
}

variable "db_instance_count" {
}

variable "db_instance_engine" {
}

variable "db_instance_class" {
}

variable "db_cluster_identifier" {
}

variable "db_engine_version" {
  type = map(any)
  default = {
    "aurora-mysql"      = "8.0.mysql_aurora.3.05.2"
    "aurora-postgresql" = "14.9"
  }
}

variable "db_username" {
}

variable "db_password" {
}

variable "db_enable_performance_insights" {
}

variable "db_parameters" {
  type = list(map(string))
}

variable "ssh_public_key" {
}

variable "mattermost_license_file" {
}

variable "block_device_type" {
  type    = string
  default = "gp3"
}

variable "block_device_sizes_agent" {
  type    = number
  default = 10
}

variable "block_device_sizes_proxy" {
  type    = number
  default = 10
}

variable "block_device_sizes_app" {
  type    = number
  default = 10
}

variable "block_device_sizes_metrics" {
  type    = number
  default = 50
}

variable "block_device_sizes_job" {
  type    = number
  default = 50
}

variable "block_device_sizes_elasticsearch" {
  type    = number
  default = 20
}

# Keycloak variables
variable "block_device_sizes_keycloak" {
  type    = number
  default = 10
}

variable "keycloak_enabled" {
  type = bool
}

variable "keycloak_instance_type" {
}

variable "keycloak_version" {
  type    = string
  default = "24.0.2"
}

variable "keycloak_development_mode" {
  type = bool
}

# Job server variables
variable "job_server_instance_count" {
}

variable "job_server_instance_type" {
}

variable "s3_bucket_dump_uri" {
}

variable "s3_external_bucket_name" {
}

variable "aws_profile" {
}

variable "aws_region" {
}

variable "aws_ami" {
}

variable "custom_tags" {
  type = map(string)
}

variable "aws_az" {
}

variable "enable_metrics_instance" {
  type = bool
}

variable "metrics_instance_type" {
}
