variable "cluster_name" {
  default = "loadtest"
}

variable "app_instance_count" {
  default = 1
}

variable "ssh_public_key" {
  default = "~/.ssh/id_rsa.pub"
}

provider "aws" {
  profile = "default"
  region  = "us-east-2"
  version = "~> 2.47"
}

resource "aws_key_pair" "key" {
  key_name   = "loadtest"
  public_key = file(var.ssh_public_key)
}

resource "aws_instance" "app_server" {
  tags = {
    Name = "${var.cluster_name}-app-${count.index}"
  }

  connection {
    # The default username for our AMI
    user = "ubuntu"
    host = "$self.public_ip"
  }

  ami           = "ami-0fc20dd1da406780b" # 18.04 LTS
  instance_type = "c5.xlarge"             # Always use c5.xlarge for consistency
  key_name      = aws_key_pair.key.id
  count         = var.app_instance_count
  vpc_security_group_ids = [
    "${aws_security_group.app.id}"
  ]
}

resource "aws_security_group" "app" {
  name        = "${var.cluster_name}-app-security-group"
  description = "App security group for loadtest cluster ${var.cluster_name}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 8065
    to_port     = 8065
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
