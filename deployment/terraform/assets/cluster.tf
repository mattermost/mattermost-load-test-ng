provider "aws" {
  profile = "mm-loadtest"
  region  = "us-east-2"
  version = "~> 2.47"
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

data "aws_subnet_ids" "selected" {
  vpc_id = "${var.es_vpc}"
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
  instance_type = var.app_instance_type
  key_name      = aws_key_pair.key.id
  count         = var.app_instance_count
  vpc_security_group_ids = [
    "${aws_security_group.app.id}",
    "${aws_security_group.app_gossip.id}"
  ]

  dynamic "root_block_device" {
    for_each = var.root_block_device
    content {
      volume_size = lookup(root_block_device.value, "volume_size", null)
      volume_type = lookup(root_block_device.value, "volume_type", null)
    }
  }

  provisioner "file" {
    source      = var.mattermost_license_file
    destination = "/home/ubuntu/mattermost.mattermost-license"
  }

  provisioner "remote-exec" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "wget --no-check-certificate -qO - https://s3-eu-west-1.amazonaws.com/deb.robustperception.io/41EFC99D.gpg | sudo apt-key add -",
      "sudo apt-get -y update",
      "sudo apt-get install -y prometheus-node-exporter",
      "wget -O mattermost-dist.tar.gz ${var.mattermost_download_url}",
      "tar xzf mattermost-dist.tar.gz",
      "sudo mv mattermost /opt/"
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
  instance_type = "t3.xlarge"
  key_name      = aws_key_pair.key.id

  vpc_security_group_ids = [
    "${aws_security_group.metrics.id}",
  ]

  dynamic "root_block_device" {
    for_each = var.root_block_device
    content {
      volume_size = lookup(root_block_device.value, "volume_size", null)
      volume_type = lookup(root_block_device.value, "volume_type", null)
    }
  }

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
      "wget https://github.com/inbucket/inbucket/releases/download/v2.1.0/inbucket_2.1.0_linux_amd64.deb",
      "sudo dpkg -i inbucket_2.1.0_linux_amd64.deb",
      "wget https://github.com/justwatchcom/elasticsearch_exporter/releases/download/v1.1.0/elasticsearch_exporter-1.1.0.linux-amd64.tar.gz",
      "sudo mkdir /opt/elasticsearch_exporter",
      "sudo tar -zxvf elasticsearch_exporter-1.1.0.linux-amd64.tar.gz -C /opt/elasticsearch_exporter --strip-components=1",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable grafana-server",
      "sudo service grafana-server start",
      "sudo systemctl enable inbucket",
      "sudo service inbucket start"
    ]
  }
}

resource "aws_instance" "proxy_server" {
  tags = {
    Name = "${var.cluster_name}-proxy"
  }
  ami                         = "ami-0fc20dd1da406780b"
  instance_type               = var.proxy_instance_type
  count                       = var.app_instance_count > 1 ? 1 : 0
  associate_public_ip_address = true
  vpc_security_group_ids = [
    "${aws_security_group.proxy.id}"
  ]
  key_name = aws_key_pair.key.id

  dynamic "root_block_device" {
    for_each = var.root_block_device
    content {
      volume_size = lookup(root_block_device.value, "volume_size", null)
      volume_type = lookup(root_block_device.value, "volume_type", null)
    }
  }

  connection {
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    host = self.public_ip
  }

  provisioner "remote-exec" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "sudo apt-get -y update",
      "sudo apt-get install -y prometheus-node-exporter",
      "sudo apt-get install -y nginx",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable nginx",
      "sudo rm -f /etc/nginx/sites-enabled/default",
      "sudo ln -fs /etc/nginx/sites-available/mattermost /etc/nginx/sites-enabled/mattermost"
    ]
  }
}

resource "aws_iam_service_linked_role" "es" {
  count = var.es_instance_count && var.es_create_role ? 1 : 0
  aws_service_name = "es.amazonaws.com"
}

resource "aws_elasticsearch_domain" "es_server" {
  tags = {
    Name = "${var.cluster_name}-es_server"
  }

  domain_name           = "${var.cluster_name}-es"
  elasticsearch_version = var.es_version

  vpc_options {
    subnet_ids = [
      element(tolist(data.aws_subnet_ids.selected.ids), 0)
    ]
    security_group_ids = ["${aws_security_group.elastic[0].id}"]
  }

  dynamic "ebs_options" {
    for_each = var.es_ebs_options
    content {
      ebs_enabled = true
      volume_type = lookup(ebs_options.value, "volume_type", "gp2")
      volume_size = lookup(ebs_options.value, "volume_size", 10)
    }
  }

  cluster_config {
    instance_type = var.es_instance_type
  }

  access_policies = <<CONFIG
  {
      "Version": "2012-10-17",
      "Statement": [
          {
              "Action": "es:*",
              "Principal": "*",
              "Effect": "Allow",
              "Resource": "arn:aws:es:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:domain/${var.cluster_name}-es/*"
          }
      ]
  }
  CONFIG

  depends_on = [
    aws_iam_service_linked_role.es,
  ]

  count = var.es_instance_count
}


resource "aws_iam_user" "s3user" {
  name  = "${var.cluster_name}-s3user"
  count = var.app_instance_count > 1 ? 1 : 0
}

resource "aws_iam_access_key" "s3key" {
  user  = aws_iam_user.s3user[0].name
  count = var.app_instance_count > 1 ? 1 : 0
}

resource "aws_s3_bucket" "s3bucket" {
  bucket = "${var.cluster_name}.s3bucket"
  acl    = "private"
  count  = var.app_instance_count > 1 ? 1 : 0
  tags = {
    Name = "${var.cluster_name}-s3bucket"
  }

  force_destroy = true
}

resource "aws_iam_user_policy" "s3userpolicy" {
  name  = "${var.cluster_name}-s3userpolicy"
  user  = aws_iam_user.s3user[0].name
  count = var.app_instance_count > 1 ? 1 : 0

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
      {
          "Effect": "Allow",
          "Action": [
              "s3:ListBucket",
              "s3:GetBucketLocation"
          ],
          "Resource": "arn:aws:s3:::${aws_s3_bucket.s3bucket[0].id}"
      },
      {
          "Action": [
              "s3:AbortMultipartUpload",
              "s3:DeleteObject",
              "s3:GetObject",
              "s3:GetObjectAcl",
              "s3:PutObject",
              "s3:PutObjectAcl"
          ],
          "Effect": "Allow",
          "Resource": "arn:aws:s3:::${aws_s3_bucket.s3bucket[0].id}/*"
      }
  ]
}
EOF
}


resource "aws_rds_cluster_instance" "cluster_instances" {
  count                      = var.db_instance_count
  identifier                 = "${var.cluster_name}-db-${count.index}"
  cluster_identifier         = aws_rds_cluster.db_cluster.id
  instance_class             = var.db_instance_class
  engine                     = var.db_instance_engine
  apply_immediately          = true
  auto_minor_version_upgrade = false
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
  instance_type = var.agent_instance_type
  key_name      = aws_key_pair.key.id
  count         = var.agent_instance_count

  vpc_security_group_ids = ["${aws_security_group.agent.id}"]

  dynamic "root_block_device" {
    for_each = var.root_block_device
    content {
      volume_size = lookup(root_block_device.value, "volume_size", null)
      volume_type = lookup(root_block_device.value, "volume_type", null)
    }
  }

  provisioner "remote-exec" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "sudo apt-get -y update",
      "sudo apt-get install -y prometheus-node-exporter",
      "wget -O tmp.tar.gz ${var.load_test_download_url}",
      "tar xzf tmp.tar.gz",
      "mv mattermost-load-test-ng* mattermost-load-test-ng",
      "rm tmp.tar.gz"
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

resource "aws_security_group" "app_gossip" {
  name        = "${var.cluster_name}-app-security-group-gossip"
  description = "App security group for gossip loadtest cluster ${var.cluster_name}"
  ingress {
    from_port       = 8074
    to_port         = 8074
    protocol        = "udp"
    security_groups = ["${aws_security_group.app.id}"]
  }
  ingress {
    from_port       = 8074
    to_port         = 8074
    protocol        = "tcp"
    security_groups = ["${aws_security_group.app.id}"]
  }
  ingress {
    from_port       = 8075
    to_port         = 8075
    protocol        = "udp"
    security_groups = ["${aws_security_group.app.id}"]
  }
  ingress {
    from_port       = 8075
    to_port         = 8075
    protocol        = "tcp"
    security_groups = ["${aws_security_group.app.id}"]
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
    from_port       = 4000
    to_port         = 4000
    protocol        = "tcp"
    self            = true
    security_groups = ["${aws_security_group.metrics.id}"]
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

resource "aws_security_group" "elastic" {
  name = "${var.cluster_name}-elastic-security-group"
  description = "Security group for elastic instance"

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = ["${aws_security_group.app.id}", "${aws_security_group.metrics.id}"]
  }

  count = var.es_instance_count
}

# We need a separate security group rule to prevent cyclic dependency between
# the app group and metrics group.
resource "aws_security_group_rule" "app-to-inbucket" {
  type                     = "ingress"
  from_port                = 2500
  to_port                  = 2500
  protocol                 = "tcp"
  security_group_id        = aws_security_group.metrics.id
  source_security_group_id = aws_security_group.app.id
}

resource "aws_security_group" "proxy" {
  name        = "${var.cluster_name}-proxy-security-group"
  description = "Proxy security group for loadtest cluster ${var.cluster_name}"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
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
