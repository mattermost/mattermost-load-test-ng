output "instances" {
  value = aws_instance.app_server[*]
}

output "dbCluster" {
  value = aws_rds_cluster_instance.cluster_instances[*]
}

output "agents" {
  value = aws_instance.loadtest_agent[*]
}

output "browserAgents" {
  value = aws_instance.loadtest_browser_agent[*]
}

output "metricsServer" {
  value = aws_instance.metrics_server
}

output "keycloakServer" {
  value = aws_instance.keycloak
}

output "openldapServer" {
  value = aws_instance.openldap
}

output "proxy" {
  value = aws_instance.proxy_server
}

output "elasticServer" {
  value     = aws_opensearch_domain.es_server
  sensitive = true
}

output "elasticRoleARN" {
  value = aws_iam_role.es_role.arn
}

output "redisServer" {
  value = aws_elasticache_cluster.redis_server
}

output "s3bucket" {
  value = aws_s3_bucket.s3bucket
}

output "s3Key" {
  value     = aws_iam_access_key.s3key
  sensitive = true
}

output "dbSecurityGroup" {
  value = aws_security_group.db
}

output "jobServers" {
  value = aws_instance.job_server[*]
}

output "efsAccessPoint" {
  value = aws_efs_access_point.shared_dir
}

output "efsFileSystem" {
  value = aws_efs_file_system.efs_shared[*]
}

output "amiUser" {
  value = var.aws_ami_user
}
