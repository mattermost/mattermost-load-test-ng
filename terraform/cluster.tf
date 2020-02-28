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
    type = "ssh"
    user = "ubuntu"
    host = self.public_ip
  }

  ami           = "ami-0fc20dd1da406780b" # 18.04 LTS
  instance_type = "c5.xlarge"             # Always use c5.xlarge for consistency
  key_name      = aws_key_pair.key.id
  count         = var.app_instance_count
  vpc_security_group_ids = [
    "${aws_security_group.app.id}"
  ]

  provisioner "file" {
    source      = var.mattermost_license_file
    destination = "/home/ubuntu/mattermost.mattermost-license"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y jq",
      "wget ${var.mattermost_download_url}",
      "tar xzf mattermost-*.tar.gz",
      "sudo mv mattermost /opt/",
      "sudo mkdir -p /opt/mattermost/data"
    ]
  }
}

resource "aws_db_instance" "db" {
  identifier          = "${var.cluster_name}-db"
  allocated_storage   = 100
  storage_type        = "io1"
  iops                = 1000
  engine              = var.db_instance_engine
  engine_version      = var.db_engine_version[var.db_instance_engine]
  instance_class      = var.db_instance_class
  name                = "${var.cluster_name}db"
  username            = var.db_username
  password            = var.db_password
  skip_final_snapshot = true
  apply_immediately   = true

  # count = var.db_instance_count # needs aws_rds_cluster to work
  vpc_security_group_ids = [
    "${aws_security_group.db.id}"
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

resource "aws_security_group" "db" {
  name = "${var.cluster_name}-db-security-group"

  ingress {
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = ["${aws_security_group.app.id}"]
  }

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = ["${aws_security_group.app.id}"]
  }
}
