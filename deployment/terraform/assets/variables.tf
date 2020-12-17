variable "cluster_name" {
}

variable "cluster_vpc_id" {
}

variable "cluster_subnet_id" {
}

variable "app_instance_count" {
}

variable "app_instance_type" {
}

variable "agent_instance_count" {
}

variable "agent_instance_type" {
}

# Elasticsearch variables

variable "es_instance_count" {
}

variable "es_instance_type" {
}

variable "es_version" {
}

variable "es_vpc" {
}

variable "es_create_role" {
}

variable "es_ebs_options" {
  default = [
    {
      volume_type = "gp2"
      volume_size = 10
    },
  ]
}

variable "proxy_instance_type" {
}

variable "db_instance_count" {
}

variable "db_instance_engine" {
}

variable "db_instance_class" {
}

variable "db_engine_version" {
  type = map
  default = {
    "aurora-mysql"      = "5.7.mysql_aurora.2.09.0"
    "aurora-postgresql" = "11.7"
    "cockroachdb"       = "20.1.2"
  }
}

variable "db_username" {
}

variable "db_password" {
}

variable "ssh_public_key" {
}

variable "mattermost_download_url" {
}

variable "mattermost_license_file" {
}

variable "load_test_download_url" {
}

variable "root_block_device" {
  default = [
    {
      volume_type = "gp2"
      volume_size = 50
    },
  ]
}
