provider "aws" {
  profile = "default"
  region  = "us-east-2"
  version = "~> 2.47"
}

resource "aws_key_pair" "key" {
  key_name   = "${var.cluster_name}-keypair"
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
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "wget --no-check-certificate -qO - https://s3-eu-west-1.amazonaws.com/deb.robustperception.io/41EFC99D.gpg | sudo apt-key add -",
      "sudo apt-get -y update",
      "sudo apt-get install -y jq",
      "sudo apt-get install -y prometheus-node-exporter",
      "wget ${var.mattermost_download_url}",
      "tar xzf mattermost-*.tar.gz",
      "sudo mv mattermost /opt/",
      "sudo mkdir -p /opt/mattermost/data"
    ]
  }
}

resource "aws_instance" "metrics_server" {
  tags = {
    Name = "${var.cluster_name}-metrics"
  }

  connection {
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    host = self.public_ip
  }

  ami           = "ami-0fc20dd1da406780b" # 18.04 LTS
  instance_type = "t2.large"
  key_name      = aws_key_pair.key.id

  vpc_security_group_ids = [
    "${aws_security_group.metrics.id}",
  ]

  provisioner "remote-exec" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "wget --no-check-certificate -qO - https://s3-eu-west-1.amazonaws.com/deb.robustperception.io/41EFC99D.gpg | sudo apt-key add -",
      "sudo apt-get -y update",
      "sudo apt-get install -y prometheus",
      "sudo systemctl enable prometheus",
      "sudo apt-get install -y adduser libfontconfig1",
      "wget https://dl.grafana.com/oss/release/grafana_6.6.2_amd64.deb",
      "sudo dpkg -i grafana_6.6.2_amd64.deb",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable grafana-server",
      "sudo service grafana-server start"
    ]
  }
}

resource "aws_rds_cluster_instance" "cluster_instances" {
  count              = var.db_instance_count
  identifier         = "${var.cluster_name}-db-${count.index}"
  cluster_identifier = aws_rds_cluster.db_cluster.id
  instance_class     = var.db_instance_class
  engine             = var.db_instance_engine
  apply_immediately  = true
}

resource "aws_rds_cluster" "db_cluster" {
  cluster_identifier  = "${var.cluster_name}-db"
  database_name       = "${var.cluster_name}db"
  master_username     = var.db_username
  master_password     = var.db_password
  skip_final_snapshot = true
  apply_immediately   = true
  engine              = var.db_instance_engine
  engine_version      = var.db_engine_version[var.db_instance_engine]

  vpc_security_group_ids = ["${aws_security_group.db.id}"]
}

resource "aws_instance" "loadtest_agent" {
  tags = {
    Name = "${var.cluster_name}-agent-${count.index}"
  }

  connection {
    type = "ssh"
    user = "ubuntu"
    host = self.public_ip
  }

  ami           = "ami-0fc20dd1da406780b"
  instance_type = "t2.medium"
  key_name      = aws_key_pair.key.id
  count         = var.loadtest_agent_count

  vpc_security_group_ids = ["${aws_security_group.agent.id}"]

  provisioner "remote-exec" {
      inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "wget --no-check-certificate -qO - https://s3-eu-west-1.amazonaws.com/deb.robustperception.io/41EFC99D.gpg | sudo apt-key add -",
      "sudo apt-get -y update",
      "sudo apt-get install -y jq",
      "wget https://dl.google.com/go/${var.go_binary_file}",
      "sudo tar -C /usr/local -xzf ${var.go_binary_file}",
      "export PATH=$PATH:/usr/local/go/bin",
      "wget https://github.com/mattermost/mattermost-load-test-ng/archive/${var.loadtest_source_code_ref}.tar.gz",
      "tar -xzf ${var.loadtest_source_code_ref}.tar.gz",
      "cp mattermost-load-test-ng-${var.loadtest_source_code_ref}/config/simplecontroller.default.json mattermost-load-test-ng-${var.loadtest_source_code_ref}/config/simplecontroller.json",
      "cp mattermost-load-test-ng-${var.loadtest_source_code_ref}/config/config.default.json mattermost-load-test-ng-${var.loadtest_source_code_ref}/config/config.json"
    ] 
  }
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
  ingress {
    from_port = 8067
    to_port   = 8067
    protocol  = "tcp"
    # Maybe restrict only from Prometheus server ?
    # But handy while taking profiles without manually ssh-ing into the server.
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port       = 9100
    to_port         = 9100
    protocol        = "tcp"
    security_groups = ["${aws_security_group.metrics.id}"]
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

resource "aws_security_group" "agent" {
  name        = "${var.cluster_name}-agent-security-group"
  description = "Loadtest agent security group for loadtest cluster ${var.cluster_name}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 4000
    to_port     = 4000
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

resource "aws_security_group" "metrics" {
  name = "${var.cluster_name}-metrics-security-group"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 9090
    to_port     = 9090
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 3000
    to_port     = 3000
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
