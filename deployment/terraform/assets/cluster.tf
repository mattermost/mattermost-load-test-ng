terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.39"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile == "" ? null : var.aws_profile
  default_tags {
    tags = merge(
      {
        "ClusterName" = var.cluster_name
      },
      var.custom_tags
    )
  }
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

locals {
  create_s3_bucket = var.app_instance_count > 1 && var.s3_external_bucket_name == "" && !var.create_efs ? 1 : 0
}

data "http" "my_public_ip" {
  url = "https://checkip.amazonaws.com"
}

data "external" "private_ip" {
  program = ["sh", "-c", "echo {\\\"ip\\\":\\\"$(hostname -i)\\\"}"]
}

locals {
  public_ip  = chomp(data.http.my_public_ip.response_body)
  private_ip = data.external.private_ip.result.ip
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
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami                  = var.aws_ami
  instance_type        = var.app_instance_type
  key_name             = aws_key_pair.key.id
  count                = var.app_instance_count
  availability_zone    = var.aws_az
  iam_instance_profile = var.app_attach_iam_profile
  subnet_id            = (length(var.cluster_subnet_ids.app) > 0) ? element(tolist(var.cluster_subnet_ids.app), count.index) : null

  vpc_security_group_ids = [
    aws_security_group.app[0].id,
    aws_security_group.app_gossip[0].id
  ]

  root_block_device {
    volume_size = var.block_device_sizes_app
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = var.mattermost_license_file
    destination = "/home/${var.aws_ami_user}/mattermost.mattermost-license"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/app.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }
}
data "aws_vpc" "selected" {
  tags = {
    Name = "Default VPC"
  }
}

data "aws_subnets" "lt_selected" {
  filter {
    name = "vpc-id"
    values = [data.aws_vpc.selected.id]
  }

  tags = {
    Name = "loadtest*"
  }
}

resource "aws_efs_file_system" "efs_shared" {
  count = var.create_efs ? 1 : 0

  tags = {
   Name = "${var.cluster_name}-shared-fs"
  }
}

resource "aws_efs_mount_target" "efs_mount" {
  for_each = (var.create_efs ? toset(data.aws_subnets.lt_selected.ids) : toset({}))
  file_system_id = aws_efs_file_system.efs_shared.0.id
  subnet_id      = each.value
  security_groups = [aws_security_group.efs[0].id]
}

resource "aws_efs_access_point" "shared_dir" {
  count = var.create_efs ? 1 : 0

  file_system_id = aws_efs_file_system.efs_shared.0.id
  tags = {
   Name = "${var.cluster_name}-shared-dir"
  }

  root_directory {
    path = "/"
  }
}

data "aws_iam_policy_document" "metrics_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "metrics_role" {
  name               = "${var.cluster_name}-metrics-role"
  assume_role_policy = data.aws_iam_policy_document.metrics_assume_role.json
}

resource "aws_iam_instance_profile" "metrics_profile" {
  name = "${var.cluster_name}-metrics_profile"
  role = aws_iam_role.metrics_role.name
}

# List of required permissions taken from
# https://github.com/nerdswords/yet-another-cloudwatch-exporter/blob/f5ddcf4323dc97034491114d4074ae672cfc411f/README.md#authentication
data "aws_iam_policy_document" "metrics_policy_document" {
  statement {
    effect    = "Allow"
    resources = ["*"]
    actions = [
      "tag:GetResources",
      "cloudwatch:GetMetricData",
      "cloudwatch:GetMetricStatistics",
      "cloudwatch:ListMetrics",
      "apigateway:GET",
      "aps:ListWorkspaces",
      "autoscaling:DescribeAutoScalingGroups",
      "dms:DescribeReplicationInstances",
      "dms:DescribeReplicationTasks",
      "ec2:DescribeTransitGatewayAttachments",
      "ec2:DescribeSpotFleetRequests",
      "shield:ListProtections",
      "storagegateway:ListGateways",
      "storagegateway:ListTagsForResource",
      "iam:ListAccountAliases",
    ]
  }
}

resource "aws_iam_role_policy" "metrics_policy" {
  name   = "${var.cluster_name}-metrics-policy"
  role   = aws_iam_role.metrics_role.name
  policy = data.aws_iam_policy_document.metrics_policy_document.json
}

resource "aws_instance" "metrics_server" {
  tags = {
    Name = "${var.cluster_name}-metrics"
  }

  connection {
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami               = var.aws_ami
  instance_type     = var.metrics_instance_type
  count             = var.enable_metrics_instance ? 1 : 0
  key_name          = aws_key_pair.key.id
  availability_zone = var.aws_az
  subnet_id         = (length(var.cluster_subnet_ids.metrics) > 0) ? element(tolist(var.cluster_subnet_ids.metrics), count.index) : null

  vpc_security_group_ids = [
    aws_security_group.metrics[0].id,
  ]

  iam_instance_profile = aws_iam_instance_profile.metrics_profile.name

  root_block_device {
    volume_size = var.block_device_sizes_metrics
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/metrics.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }

}

resource "aws_instance" "proxy_server" {
  tags = {
    Name = "${var.cluster_name}-proxy-${count.index}"
  }

  ami                         = var.aws_ami
  instance_type               = var.proxy_instance_type
  count                       = var.proxy_instance_count
  associate_public_ip_address = var.proxy_allocate_public_ip_address
  availability_zone           = var.aws_az
  subnet_id                   = (length(var.cluster_subnet_ids.proxy) > 0) ? element(tolist(var.cluster_subnet_ids.proxy), count.index) : null

  vpc_security_group_ids = [
    aws_security_group.proxy[0].id
  ]
  key_name = aws_key_pair.key.id

  root_block_device {
    volume_size = var.block_device_sizes_proxy
    volume_type = var.block_device_type
  }

  connection {
    type = "ssh"
    user = var.aws_ami_user
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/proxy.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }

}

resource "aws_iam_user" "s3user" {
  name  = "${var.cluster_name}-s3user"
  count = local.create_s3_bucket
}

resource "aws_iam_access_key" "s3key" {
  user  = aws_iam_user.s3user[0].name
  count = local.create_s3_bucket
}

resource "aws_s3_bucket" "s3bucket" {
  bucket = "${var.cluster_name}.s3bucket"
  count  = local.create_s3_bucket
  tags = {
    Name = "${var.cluster_name}-s3bucket"
  }

  force_destroy = true
}

resource "aws_iam_user_policy" "s3userpolicy" {
  name  = "${var.cluster_name}-s3userpolicy"
  user  = aws_iam_user.s3user[0].name
  count = local.create_s3_bucket

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
      {
          "Effect": "Allow",
          "Action": [
              "s3:ListBucket",
              "s3:ListBucketMultiPartUploads",
              "s3:DeleteBucket",
              "s3:GetBucketLocation"
          ],
          "Resource": "arn:aws:s3:::${aws_s3_bucket.s3bucket[0].id}"
      },
      {
          "Action": [
              "s3:AbortMultipartUpload",
              "s3:ListMultipartUploadParts",
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

resource "aws_elasticache_subnet_group" "redis" {
  name       = "${var.cluster_name}-redis-subnet-group"
  subnet_ids = tolist(var.cluster_subnet_ids.redis)
  count      = var.redis_enabled && length(var.cluster_subnet_ids.redis) > 1 ? 1 : 0

  tags = {
    Name = "${var.cluster_name}-redis-subnet-group-${count.index}"
  }
}


resource "aws_elasticache_cluster" "redis_server" {
  cluster_id           = "${var.cluster_name}-redis"
  engine               = "redis"
  node_type            = var.redis_node_type
  count                = var.redis_enabled ? 1 : 0
  num_cache_nodes      = 1
  parameter_group_name = var.redis_param_group_name
  engine_version       = var.redis_engine_version
  port                 = 6379
  security_group_ids   = [aws_security_group.redis[0].id]
  availability_zone    = var.aws_az
  subnet_group_name    = var.redis_enabled && length(var.cluster_subnet_ids.redis) > 1 ? aws_elasticache_subnet_group.redis[0].name : ""
}

resource "aws_db_subnet_group" "db" {
  name       = "${var.cluster_name}-db-subnet-group"
  subnet_ids = tolist(var.cluster_subnet_ids.database)
  count      = var.db_instance_count > 0 && length(var.cluster_subnet_ids.database) > 1 ? 1 : 0

  tags = {
    Name = "${var.cluster_name}-db-subnet-group-${count.index}"
  }
}

resource "aws_rds_cluster" "db_cluster" {
  tags = {
    Name = "${var.cluster_name}-db-cluster"
  }

  count                  = var.app_instance_count > 0 && var.db_instance_count > 0 && var.db_cluster_identifier == "" ? 1 : 0
  cluster_identifier     = var.db_cluster_identifier != "" ? "" : "${var.cluster_name}-db"
  database_name          = "${var.cluster_name}db"
  master_username        = var.db_username
  master_password        = var.db_password
  skip_final_snapshot    = true
  apply_immediately      = true
  engine                 = var.db_instance_engine
  engine_version         = var.db_engine_version[var.db_instance_engine]
  db_subnet_group_name   = var.app_instance_count > 0 && var.db_instance_count > 0 && length(var.cluster_subnet_ids.database) > 1 ? aws_db_subnet_group.db[0].name : ""
  vpc_security_group_ids = [aws_security_group.db[0].id]
}

resource "aws_rds_cluster_instance" "cluster_instances" {
  tags = {
    Name = "${var.cluster_name}-db-${count.index}"
  }

  count                        = var.app_instance_count > 0 ? var.db_instance_count : 0
  identifier                   = "${var.cluster_name}-db-${count.index}"
  cluster_identifier           = var.db_cluster_identifier != "" ? var.db_cluster_identifier : aws_rds_cluster.db_cluster[0].id
  instance_class               = var.db_instance_class
  engine                       = var.db_instance_engine
  apply_immediately            = true
  auto_minor_version_upgrade   = false
  performance_insights_enabled = var.db_enable_performance_insights
  db_parameter_group_name      = length(var.db_parameters) > 0 ? aws_db_parameter_group.db_params_group.name : ""
  availability_zone            = var.aws_az
  db_subnet_group_name         = var.app_instance_count > 0 && var.db_instance_count > 0 && length(var.cluster_subnet_ids.database) > 1 ? aws_db_subnet_group.db[0].name : ""
}

resource "aws_db_parameter_group" "db_params_group" {
  name_prefix = "${var.cluster_name}-db-pg"
  family      = var.db_instance_engine == "aurora-mysql" ? "aurora-mysql8.0" : "aurora-postgresql14"
  dynamic "parameter" {
    for_each = var.db_parameters
    content {
      name         = parameter.value["name"]
      value        = parameter.value["value"]
      apply_method = parameter.value["apply_method"]
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_instance" "loadtest_agent" {
  tags = {
    Name = "${var.cluster_name}-agent-${count.index}"
  }

  connection {
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami           = var.aws_ami
  instance_type = var.agent_instance_type
  key_name      = aws_key_pair.key.id
  count         = var.agent_instance_count
  subnet_id     = (length(var.cluster_subnet_ids.agent) > 0) ? element(tolist(var.cluster_subnet_ids.agent), count.index) : null

  associate_public_ip_address = var.agent_allocate_public_ip_address
  availability_zone           = var.aws_az

  vpc_security_group_ids = [aws_security_group.agent.id]

  root_block_device {
    volume_size = var.block_device_sizes_agent
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/agent.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }

}

resource "aws_instance" "loadtest_browser_agent" {
  tags = {
    Name = "${var.cluster_name}-browser-agent-${count.index}"
  }

  connection {
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami           = var.aws_ami
  instance_type = var.browser_agent_instance_type
  key_name      = aws_key_pair.key.id
  count         = var.browser_agent_instance_count
  subnet_id     = (length(var.cluster_subnet_ids.agent) > 0) ? element(tolist(var.cluster_subnet_ids.agent), count.index) : null

  associate_public_ip_address = var.agent_allocate_public_ip_address
  availability_zone           = var.aws_az

  vpc_security_group_ids = [aws_security_group.agent.id]

  root_block_device {
    volume_size = var.block_device_sizes_agent
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/agent.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }

}

resource "aws_security_group" "app" {
  count       = var.app_instance_count > 0 ? 1 : 0
  name        = "${var.cluster_name}-app-security-group"
  description = "App security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "app_ssh_public" {
  count             = var.app_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.app[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.public_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "app_ssh_private" {
  count             = var.app_instance_count > 0 && local.private_ip != "" ? 1 : 0
  security_group_id = aws_security_group.app[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.private_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "app_web" {
  count             = var.app_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.app[0].id
  from_port         = 8065
  to_port           = 8065
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "app_metrics" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.app[0].id
  from_port         = 8067
  to_port           = 8067
  ip_protocol       = "tcp"
  # Maybe restrict only from Prometheus server ?
  # But handy while taking profiles without manually ssh-ing into the server.
  cidr_ipv4 = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "app_nodeexporter" {
  count                        = var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.app[0].id
  from_port                    = 9100
  to_port                      = 9100
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_vpc_security_group_ingress_rule" "app_netpeek" {
  count                        = var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.app[0].id
  from_port                    = 9045
  to_port                      = 9045
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_vpc_security_group_egress_rule" "app_egress" {
  count             = var.app_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.app[0].id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_security_group" "app_gossip" {
  count       = var.app_instance_count > 0 ? 1 : 0
  name        = "${var.cluster_name}-app-security-group-gossip"
  description = "App security group for gossip loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "app_gossip_8074_udp" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.app_gossip[0].id
  from_port                    = 8074
  to_port                      = 8074
  ip_protocol                  = "udp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_ingress_rule" "app_gossip_8074_tcp" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.app_gossip[0].id
  from_port                    = 8074
  to_port                      = 8074
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_ingress_rule" "app_gossip_8075_udp" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.app_gossip[0].id
  from_port                    = 8075
  to_port                      = 8075
  ip_protocol                  = "udp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_ingress_rule" "app_gossip_8075_tcp" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.app_gossip[0].id
  from_port                    = 8075
  to_port                      = 8075
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_egress_rule" "app_gossip_egress" {
  count             = var.app_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.app_gossip[0].id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}


resource "aws_security_group" "db" {
  count  = var.app_instance_count > 0 ? 1 : 0
  name   = "${var.cluster_name}-db-security-group"
  vpc_id = var.cluster_vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "db_msyql" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.db[0].id
  from_port                    = 3306
  to_port                      = 3306
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_ingress_rule" "db_postgres" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.db[0].id
  from_port                    = 5432
  to_port                      = 5432
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_security_group" "agent" {
  name        = "${var.cluster_name}-agent-security-group"
  description = "Loadtest agent security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "agent_ssh_public" {
  security_group_id = aws_security_group.agent.id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.public_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "agent_ssh_private" {
  count             = local.private_ip != "" ? 1 : 0
  security_group_id = aws_security_group.agent.id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.private_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "agent_api" {
  security_group_id = aws_security_group.agent.id
  from_port         = 4000
  to_port           = 4000
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "agent_egress" {
  security_group_id = aws_security_group.agent.id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "agent_metrics-to-prometheus" {
  count                        = var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.agent.id
  from_port                    = 4000
  to_port                      = 4000
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_vpc_security_group_ingress_rule" "agent-node-exporter" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.agent.id
  from_port                    = 9100
  to_port                      = 9100
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group" "metrics" {
  count  = var.enable_metrics_instance ? 1 : 0
  name   = "${var.cluster_name}-metrics-security-group"
  vpc_id = var.cluster_vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "metrics_ssh_publicip" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.public_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "metrics_ssh_privateip" {
  count             = var.enable_metrics_instance && local.private_ip != "" ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.private_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "metrics_prometheus" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 9090
  to_port           = 9090
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}


resource "aws_vpc_security_group_ingress_rule" "metrics_cloudwatchexporter" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 9106
  to_port           = 9106
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "metrics_grafana" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 3000
  to_port           = 3000
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "metrics_pyroscope" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 4040
  to_port           = 4040
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "metrics_loki" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  from_port         = 3100
  to_port           = 3100
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "metrics_egress" {
  count             = var.enable_metrics_instance ? 1 : 0
  security_group_id = aws_security_group.metrics[0].id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_security_group" "redis" {
  name        = "${var.cluster_name}-redis-security-group"
  description = "Security group for redis instance"
  vpc_id      = var.cluster_vpc_id

  count = var.redis_enabled ? 1 : 0
}


resource "aws_vpc_security_group_ingress_rule" "redis_app" {
  count                        = var.redis_enabled ? 1 : 0
  security_group_id            = aws_security_group.redis[0].id
  from_port                    = 6379
  to_port                      = 6379
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_ingress_rule" "redis_metrics" {
  count                        = var.redis_enabled && var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.redis[0].id
  from_port                    = 6379
  to_port                      = 6379
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group" "elastic" {
  name        = "${var.cluster_name}-elastic-security-group"
  description = "Security group for elastic instance"
  vpc_id      = var.cluster_vpc_id

  count = var.es_instance_count > 0 ? 1 : 0
}

resource "aws_vpc_security_group_ingress_rule" "elastic_app" {
  count                        = var.es_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.elastic[0].id
  from_port                    = 443
  to_port                      = 443
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_vpc_security_group_ingress_rule" "elastic_metrics" {
  count                        = var.es_instance_count > 0 && var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.elastic[0].id
  from_port                    = 443
  to_port                      = 443
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}


# We need a separate security group rule to prevent cyclic dependency between
# the app group and metrics group.
resource "aws_vpc_security_group_ingress_rule" "app_to-inbucket" {
  count                        = var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.metrics[0].id
  from_port                    = 2500
  to_port                      = 2500
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

resource "aws_security_group" "proxy" {
  count       = var.proxy_instance_count > 0 ? 1 : 0
  name        = "${var.cluster_name}-proxy-security-group"
  description = "Proxy security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "proxy_web" {
  count             = var.proxy_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.proxy[0].id
  from_port         = 80
  to_port           = 80
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "proxy_ssh_publicip" {
  count             = var.proxy_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.proxy[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.public_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "proxy_ssh_privateip" {
  count             = var.proxy_instance_count > 0 && local.private_ip != "" ? 1 : 0
  security_group_id = aws_security_group.proxy[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.private_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "proxy_metrics" {
  count                        = var.proxy_instance_count > 0 && var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.proxy[0].id
  from_port                    = 9100
  to_port                      = 9100
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_vpc_security_group_egress_rule" "proxy_egress" {
  count             = var.proxy_instance_count > 0 ? 1 : 0
  security_group_id = aws_security_group.proxy[0].id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_instance" "job_server" {
  tags = {
    Name = "${var.cluster_name}-job-server-${count.index}"
  }

  connection {
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami               = var.aws_ami
  instance_type     = var.job_server_instance_type
  key_name          = aws_key_pair.key.id
  count             = var.job_server_instance_count
  availability_zone = var.aws_az
  subnet_id         = (length(var.cluster_subnet_ids.job) > 0) ? element(tolist(var.cluster_subnet_ids.job), count.index) : null

  vpc_security_group_ids = [
    aws_security_group.app[0].id,
  ]

  root_block_device {
    volume_size = var.block_device_sizes_job
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = var.mattermost_license_file
    destination = "/home/${var.aws_ami_user}/mattermost.mattermost-license"
  }


  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/job.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }
}

locals {
  profile_flag = var.aws_profile == "" ? "" : "--profile ${var.aws_profile}"
}

resource "null_resource" "s3_dump" {
  count = (var.app_instance_count > 1 && var.s3_bucket_dump_uri != "" && var.s3_external_bucket_name == "" && !var.create_efs) ? 1 : 0

  provisioner "local-exec" {
    command = "aws ${local.profile_flag} s3 cp ${var.s3_bucket_dump_uri} s3://${aws_s3_bucket.s3bucket[0].id} --recursive"
  }
}

// Keycloak
resource "aws_instance" "keycloak" {
  tags = {
    Name = "${var.cluster_name}-keycloak"
  }

  connection {
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami               = var.aws_ami
  instance_type     = var.keycloak_instance_type
  count             = var.keycloak_enabled ? 1 : 0
  key_name          = aws_key_pair.key.id
  availability_zone = var.aws_az
  subnet_id         = (length(var.cluster_subnet_ids.keycloak) > 0) ? element(tolist(var.cluster_subnet_ids.keycloak), count.index) : null

  vpc_security_group_ids = [
    aws_security_group.keycloak[0].id,
  ]

  root_block_device {
    volume_size = var.block_device_sizes_keycloak
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/keycloak.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "cd /tmp",
      "chmod +x /tmp/common.sh",
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh",
    ]
  }
}

resource "aws_security_group" "keycloak" {
  count       = var.keycloak_enabled ? 1 : 0
  name        = "${var.cluster_name}-keycloak-security-group"
  description = "KeyCloak security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id

}

resource "aws_vpc_security_group_egress_rule" "keycloak_egress" {
  count             = var.keycloak_enabled ? 1 : 0
  security_group_id = aws_security_group.keycloak[0].id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "keycloak_ssh_publicip" {
  count             = var.keycloak_enabled ? 1 : 0
  security_group_id = aws_security_group.keycloak[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.public_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "keycloak_ssh_privateip" {
  count             = var.keycloak_enabled && local.private_ip != "" ? 1 : 0
  security_group_id = aws_security_group.keycloak[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.private_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "keycloak_https" {
  count             = var.keycloak_enabled ? 1 : 0
  security_group_id = aws_security_group.keycloak[0].id
  from_port         = 8443
  to_port           = 8443
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "keycloak_http" {
  count             = var.keycloak_enabled ? 1 : 0
  security_group_id = aws_security_group.keycloak[0].id
  from_port         = 8080
  to_port           = 8080
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_security_group" "efs" {
  count       = var.create_efs ? 1 : 0
  name        = "${var.cluster_name}-efs-security-group"
  description = "EFS security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id

  # Remove the app security group reference from here
  ingress {
    from_port   = 2049
    to_port     = 2049
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]  # Or use a more restricted CIDR range for your VPC
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1" # Allow all outbound traffic
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "efs-sg"
  }
}

resource "aws_security_group_rule" "efs-from-app" {
  count                    = var.create_efs && var.app_instance_count > 0 ? 1 : 0
  type                     = "ingress"
  from_port                = 2049
  to_port                  = 2049
  protocol                 = "tcp"
  security_group_id        = aws_security_group.efs[0].id
  source_security_group_id = aws_security_group.app[0].id
}

resource "aws_security_group_rule" "app-to-efs" {
  count                    = var.create_efs && var.app_instance_count > 0 ? 1 : 0
  type                     = "ingress"
  from_port                = 2049
  to_port                  = 2049
  protocol                 = "tcp"
  security_group_id        = aws_security_group.app[0].id
  source_security_group_id = aws_security_group.efs[0].id
}