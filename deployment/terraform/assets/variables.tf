variable "cluster_name" {
}

variable "app_instance_count" {
}

variable "app_instance_type" {
}

variable "agent_instance_count" {
}

variable "agent_instance_type" {
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
    "aurora-mysql"      = "5.7.mysql_aurora.2.03.2"
    "aurora-postgresql" = "9.6.8"
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
