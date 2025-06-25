// OpenLDAP Server
resource "aws_instance" "openldap" {
  tags = {
    Name = "${var.cluster_name}-openldap"
  }

  connection {
    type = "ssh"
    host = var.connection_type == "public" ? self.public_ip : self.private_ip
    user = var.aws_ami_user
  }

  ami               = var.aws_ami
  instance_type     = var.openldap_instance_type
  count             = var.openldap_enabled ? 1 : 0
  key_name          = aws_key_pair.key.id
  availability_zone = var.aws_az
  subnet_id         = (length(var.cluster_subnet_ids.openldap) > 0) ? element(tolist(var.cluster_subnet_ids.openldap), count.index) : null

  vpc_security_group_ids = [
    aws_security_group.openldap[0].id,
  ]

  root_block_device {
    volume_size = var.block_device_sizes_openldap
    volume_type = var.block_device_type
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "provisioners/${var.operating_system_kind}/openldap.sh"
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

resource "aws_security_group" "openldap" {
  count       = var.openldap_enabled ? 1 : 0
  name        = "${var.cluster_name}-openldap-security-group"
  description = "OpenLDAP security group for loadtest cluster ${var.cluster_name}"
  vpc_id      = var.cluster_vpc_id
}

resource "aws_vpc_security_group_egress_rule" "openldap_egress" {
  count             = var.openldap_enabled ? 1 : 0
  security_group_id = aws_security_group.openldap[0].id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "openldap_ssh_publicip" {
  count             = var.openldap_enabled ? 1 : 0
  security_group_id = aws_security_group.openldap[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.public_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "openldap_ssh_privateip" {
  count             = var.openldap_enabled && local.private_ip != "" ? 1 : 0
  security_group_id = aws_security_group.openldap[0].id
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
  cidr_ipv4         = "${local.private_ip}/32"
}

resource "aws_vpc_security_group_ingress_rule" "openldap_ldap" {
  count             = var.openldap_enabled ? 1 : 0
  security_group_id = aws_security_group.openldap[0].id
  from_port         = 389
  to_port           = 389
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "openldap_ldaps" {
  count             = var.openldap_enabled ? 1 : 0
  security_group_id = aws_security_group.openldap[0].id
  from_port         = 636
  to_port           = 636
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

# Allow app servers to connect to LDAP (port 389)
resource "aws_vpc_security_group_ingress_rule" "openldap_app_ldap" {
  count                        = var.openldap_enabled && var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.openldap[0].id
  from_port                    = 389
  to_port                      = 389
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

# Allow app servers to connect to LDAPS (port 636)
resource "aws_vpc_security_group_ingress_rule" "openldap_app_ldaps" {
  count                        = var.openldap_enabled && var.app_instance_count > 0 ? 1 : 0
  security_group_id            = aws_security_group.openldap[0].id
  from_port                    = 636
  to_port                      = 636
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.app[0].id
}

# Allow metrics server to connect to node exporter (port 9100)
resource "aws_vpc_security_group_ingress_rule" "openldap_metrics_node_exporter" {
  count                        = var.openldap_enabled && var.enable_metrics_instance ? 1 : 0
  security_group_id            = aws_security_group.openldap[0].id
  from_port                    = 9100
  to_port                      = 9100
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.metrics[0].id
}
