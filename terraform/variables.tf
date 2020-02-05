variable "cluster_name" {
  default = "loadtest"
}

variable "app_instance_count" {
  default = 1
}

variable "db_instance_count" {
  default = 1
}

variable "db_instance_engine" {
  default = "mysql"
}

variable "db_engine_version" {
  type = map
  default = {
    "mysql"    = "5.7"
    "postgres" = "9.6"
  }
}

variable "db_username" {
  default = "mmuser"
}

variable "db_password" {
  default = "mostest"
}

variable "ssh_public_key" {
  default = "~/.ssh/id_rsa.pub"
}
