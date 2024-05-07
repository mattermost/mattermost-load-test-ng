resource "aws_iam_service_linked_role" "es" {
  count            = var.es_instance_count > 0 && var.es_create_role ? 1 : 0
  aws_service_name = "es.amazonaws.com"
}

data "aws_iam_policy_document" "es_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["es.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "es_role" {
  name               = "${var.cluster_name}-es-role"
  assume_role_policy = data.aws_iam_policy_document.es_assume_role.json
}

data "aws_iam_policy_document" "es_policy_document" {
  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = ["arn:aws:s3:::${var.es_snapshot_repository}"]
  }

  statement {
    effect = "Allow"
    # Put and Delete are needed to register the repository since the client uploads and then deletes a test object to ensure everything's working as expected
    actions   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
    resources = ["arn:aws:s3:::${var.es_snapshot_repository}/*"]
  }
}

resource "aws_iam_policy" "es_policy" {
  name   = "${var.cluster_name}-es-policy"
  policy = data.aws_iam_policy_document.es_policy_document.json
}

resource "aws_iam_role_policy_attachment" "es_attach" {
  role       = aws_iam_role.es_role.name
  policy_arn = aws_iam_policy.es_policy.arn
}

resource "aws_cloudwatch_log_group" "es_log_group" {
  name = "${var.cluster_name}-log-group"
}

data "aws_iam_policy_document" "es_log_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["es.amazonaws.com"]
    }

    actions = [
      "logs:PutLogEvents",
      "logs:PutLogEventsBatch",
      "logs:CreateLogStream",
    ]

    resources = ["arn:aws:logs:*"]
  }
}
resource "aws_cloudwatch_log_resource_policy" "es_log_cloudwatch_policy" {
  policy_name     = "${var.cluster_name}-cloudwatch-log-policy"
  policy_document = data.aws_iam_policy_document.es_log_policy.json
}

resource "aws_opensearch_domain" "es_server" {
  tags = {
    Name = "${var.cluster_name}-es_server"
  }

  domain_name    = "${var.cluster_name}-es"
  engine_version = var.es_version

  vpc_options {
    subnet_ids = [
      element(tolist(data.aws_subnets.selected.ids), 0)
    ]
    security_group_ids = [aws_security_group.elastic[0].id]
  }


  ebs_options {
    ebs_enabled = true
    volume_type = var.block_device_type
    volume_size = var.block_device_sizes_elasticsearch
  }

  cluster_config {
    instance_count = var.es_instance_count
    instance_type  = var.es_instance_type
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

  log_publishing_options {
    cloudwatch_log_group_arn = aws_cloudwatch_log_group.es_log_group.arn
    log_type                 = "ES_APPLICATION_LOGS"
  }

  advanced_security_options {
    enabled                        = false
    anonymous_auth_enabled         = true
    internal_user_database_enabled = true
    master_user_options {
      master_user_name     = "master_user_name"
      master_user_password = "Master_user_passw0rd"
    }
  }

  encrypt_at_rest {
    enabled = true
  }

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }

  node_to_node_encryption {
    enabled = true
  }

  count = var.es_instance_count > 0 ? 1 : 0
}
