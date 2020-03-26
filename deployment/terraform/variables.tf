variable "cluster_name" {
}

variable "app_instance_count" {
}

variable "loadtest_agent_count" {
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
    "aurora-postgresql" = "9.6.3"
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

variable "go_version" {
}

variable "loadtest_source_code_ref" {
}
