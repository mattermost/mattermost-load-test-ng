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
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    # BRANCH: Use private IP (using private subnet)
    host = self.private_ip
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
    destination = "/home/ubuntu/mattermost.mattermost-license"
  }

  provisioner "remote-exec" {
    script = "provisioners/app.sh"
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
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    # BRANCH: Use private IP (using private subnet)
    host = self.private_ip
  }

  ami               = var.aws_ami
  instance_type     = var.metrics_instance_type
  count             = var.app_instance_count > 0 ? 1 : 0
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

  provisioner "remote-exec" {
    script = "provisioners/metrics.sh"
  }
}

resource "aws_instance" "proxy_server" {
  tags = {
    Name = "${var.cluster_name}-proxy-${count.index}"
  }

  ami                         = var.aws_ami
  instance_type               = var.proxy_instance_type
  count                       = var.proxy_instance_count
  associate_public_ip_address = true
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
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    # BRANCH: Use private IP (using private subnet)
    host = self.private_ip
  }

  provisioner "remote-exec" {
    script = "provisioners/proxy.sh"
  }
}

resource "aws_iam_user" "s3user" {
  name  = "${var.cluster_name}-s3user"
  # BRANCH: count = var.app_instance_count > 1 && var.s3_external_bucket_name == "" ? 1 : 0
  count = 0
}

resource "aws_iam_access_key" "s3key" {
  user  = aws_iam_user.s3user[0].name
  # BRANCH: count = var.app_instance_count > 1 && var.s3_external_bucket_name == "" ? 1 : 0
  count = 0
}

resource "aws_s3_bucket" "s3bucket" {
  bucket = "${var.cluster_name}.s3bucket"
  # BRANCH: count  = var.app_instance_count > 1 && var.s3_external_bucket_name == "" ? 1 : 0
  count = 0
  tags = {
    Name = "${var.cluster_name}-s3bucket"
  }

  force_destroy = true
}

resource "aws_iam_user_policy" "s3userpolicy" {
  name  = "${var.cluster_name}-s3userpolicy"
  user  = aws_iam_user.s3user[0].name
  # BRANCH: count = var.app_instance_count > 1 && var.s3_external_bucket_name == "" ? 1 : 0
  count = 0

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
  db_parameter_group_name      = length(var.db_parameters) > 0 ? "${var.cluster_name}-db-pg" : ""
  availability_zone            = var.aws_az
  db_subnet_group_name         = var.app_instance_count > 0 && var.db_instance_count > 0 && length(var.cluster_subnet_ids.database) > 1 ? aws_db_subnet_group.db[0].name : ""
}

resource "aws_db_parameter_group" "db_params_group" {
  name   = "${var.cluster_name}-db-pg"
  family = var.db_instance_engine == "aurora-mysql" ? "aurora-mysql8.0" : "aurora-postgresql14"
  dynamic "parameter" {
    for_each = var.db_parameters
    content {
      name         = parameter.value["name"]
      value        = parameter.value["value"]
      apply_method = parameter.value["apply_method"]
    }
  }
}

resource "aws_instance" "loadtest_agent" {
  tags = {
    Name = "${var.cluster_name}-agent-${count.index}"
  }

  connection {
    type = "ssh"
    user = "ubuntu"
    # BRANCH: Use private IP (using private subnet)
    host = self.private_ip
  }

  ami           = var.aws_ami
  instance_type = var.agent_instance_type
  key_name      = aws_key_pair.key.id
  count         = var.agent_instance_count
  subnet_id     = (length(var.cluster_subnet_ids.agent) > 0) ? element(tolist(var.cluster_subnet_ids.agent), count.index) : null

  associate_public_ip_address = true
  availability_zone           = var.aws_az

  vpc_security_group_ids = [aws_security_group.agent.id]

  root_block_device {
    volume_size = var.block_device_sizes_agent
    volume_type = var.block_device_type
  }

  provisioner "remote-exec" {
    script = "provisioners/agent.sh"
  }
}

resource "aws_security_group" "app" {
  count       = var.app_instance_count > 0 ? 1 : 0
  name        = "${var.cluster_name}-app-security-group"
  description = "App security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = local.private_ip != "" ? ["${local.public_ip}/32", "${local.private_ip}/32"] : ["${local.public_ip}/32"]
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
    security_groups = [aws_security_group.metrics[0].id]
  }
  # netpeek metrics
  ingress {
    from_port       = 9045
    to_port         = 9045
    protocol        = "tcp"
    security_groups = [aws_security_group.metrics[0].id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "app_gossip" {
  count       = var.app_instance_count > 0 ? 1 : 0
  name        = "${var.cluster_name}-app-security-group-gossip"
  description = "App security group for gossip loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id

  ingress {
    from_port       = 8074
    to_port         = 8074
    protocol        = "udp"
    security_groups = [aws_security_group.app[0].id]
  }
  ingress {
    from_port       = 8074
    to_port         = 8074
    protocol        = "tcp"
    security_groups = [aws_security_group.app[0].id]
  }
  ingress {
    from_port       = 8075
    to_port         = 8075
    protocol        = "udp"
    security_groups = [aws_security_group.app[0].id]
  }
  ingress {
    from_port       = 8075
    to_port         = 8075
    protocol        = "tcp"
    security_groups = [aws_security_group.app[0].id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}


resource "aws_security_group" "db" {
  count  = var.app_instance_count > 0 ? 1 : 0
  name   = "${var.cluster_name}-db-security-group"
  vpc_id = var.cluster_vpc_id

  ingress {
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [aws_security_group.app[0].id]
  }

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app[0].id]
  }
}

resource "aws_security_group" "agent" {
  name        = "${var.cluster_name}-agent-security-group"
  description = "Loadtest agent security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id
}

resource "aws_security_group_rule" "agent-ssh" {
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = local.private_ip != "" ? ["${local.public_ip}/32", "${local.private_ip}/32"] : ["${local.public_ip}/32"]
  security_group_id = aws_security_group.agent.id
}

resource "aws_security_group_rule" "agent-api" {
  type              = "ingress"
  from_port         = 4000
  to_port           = 4000
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.agent.id
}

resource "aws_security_group_rule" "agent-egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.agent.id
}

resource "aws_security_group_rule" "agent-metrics-to-prometheus" {
  count                    = var.app_instance_count > 0 ? 1 : 0
  type                     = "ingress"
  from_port                = 4000
  to_port                  = 4000
  protocol                 = "tcp"
  security_group_id        = aws_security_group.agent.id
  source_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group_rule" "agent-node-exporter" {
  count                    = var.app_instance_count > 0 ? 1 : 0
  type                     = "ingress"
  from_port                = 9100
  to_port                  = 9100
  protocol                 = "tcp"
  security_group_id        = aws_security_group.agent.id
  source_security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group" "metrics" {
  count  = var.app_instance_count > 0 ? 1 : 0
  name   = "${var.cluster_name}-metrics-security-group"
  vpc_id = var.cluster_vpc_id
}

resource "aws_security_group_rule" "metrics-ssh" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = local.private_ip != "" ? ["${local.public_ip}/32", "${local.private_ip}/32"] : ["${local.public_ip}/32"]
  security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group_rule" "metrics-prometheus" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "ingress"
  from_port         = 9090
  to_port           = 9090
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.metrics[0].id
}


resource "aws_security_group_rule" "metrics-cloudwatchexporter" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "ingress"
  from_port         = 9106
  to_port           = 9106
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group_rule" "metrics-grafana" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "ingress"
  from_port         = 3000
  to_port           = 3000
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group_rule" "metrics-pyroscope" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "ingress"
  from_port         = 4040
  to_port           = 4040
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group_rule" "metrics-loki" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "ingress"
  from_port         = 3100
  to_port           = 3100
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group_rule" "metrics-egress" {
  count             = var.app_instance_count > 0 ? 1 : 0
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.metrics[0].id
}

resource "aws_security_group" "redis" {
  name        = "${var.cluster_name}-redis-security-group"
  description = "Security group for redis instance"
  vpc_id      = var.cluster_vpc_id

  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.app[0].id, aws_security_group.metrics[0].id]
  }

  count = 1
}

resource "aws_security_group" "elastic" {
  name        = "${var.cluster_name}-elastic-security-group"
  description = "Security group for elastic instance"
  vpc_id      = var.cluster_vpc_id

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.app[0].id, aws_security_group.metrics[0].id]
  }

  count = var.es_instance_count > 0 ? 1 : 0
}

# We need a separate security group rule to prevent cyclic dependency between
# the app group and metrics group.
resource "aws_security_group_rule" "app-to-inbucket" {
  count                    = var.app_instance_count > 0 ? 1 : 0
  type                     = "ingress"
  from_port                = 2500
  to_port                  = 2500
  protocol                 = "tcp"
  security_group_id        = aws_security_group.metrics[0].id
  source_security_group_id = aws_security_group.app[0].id
}

resource "aws_security_group" "proxy" {
  count       = var.proxy_instance_count
  name        = "${var.cluster_name}-proxy-security-group"
  description = "Proxy security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id

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
    cidr_blocks = local.private_ip != "" ? ["${local.public_ip}/32", "${local.private_ip}/32"] : ["${local.public_ip}/32"]
  }

  ingress {
    from_port       = 9100
    to_port         = 9100
    protocol        = "tcp"
    security_groups = [aws_security_group.metrics[0].id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "job_server" {
  tags = {
    Name = "${var.cluster_name}-job-server-${count.index}"
  }

  connection {
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    # BRANCH: Use private IP (using private subnet)
    host = self.private_ip
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
    destination = "/home/ubuntu/mattermost.mattermost-license"
  }

  provisioner "remote-exec" {
    script = "provisioners/job.sh"
  }
}

locals {
  profile_flag = var.aws_profile == "" ? "" : "--profile ${var.aws_profile}"
}

resource "null_resource" "s3_dump" {
  count = (var.app_instance_count > 1 && var.s3_bucket_dump_uri != "" && var.s3_external_bucket_name == "") ? 1 : 0

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
    # The default username for our AMI
    type = "ssh"
    user = "ubuntu"
    host = self.public_ip
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
    source      = "provisioners/keycloak.sh"
    destination = "/tmp/provisioner.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/provisioner.sh",
      "/tmp/provisioner.sh ${var.keycloak_version}",
    ]
  }
}

resource "aws_security_group" "keycloak" {
  count       = var.keycloak_enabled ? 1 : 0
  name        = "${var.cluster_name}-keycloak-security-group"
  description = "KeyCloak security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = local.private_ip != "" ? ["${local.public_ip}/32", "${local.private_ip}/32"] : ["${local.public_ip}/32"]
  }

  // To access keycloak
  ingress {
    from_port   = 8443
    to_port     = 8443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
