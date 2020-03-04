variable "cluster_name" {
}

variable "app_instance_count" {
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
    "mysql"    = "5.7"
    "postgres" = "9.6"
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
