output "instances" {
  value = aws_instance.app_server[*]
}

output "dbCluster" {
  value = aws_rds_cluster_endpoint.cluster_endpoints[*]
}

output "agents" {
  value = aws_instance.loadtest_agent[*]
}

output "metricsServer" {
  value = aws_instance.metrics_server
}

output "keycloakServer" {
  value = aws_instance.keycloak
}

output "keycloakDatabaseCluster" {
  value = aws_rds_cluster_endpoint.keycloak_cluster_endpoints[*]
}

output "proxy" {
  value = aws_instance.proxy_server
}

output "elasticServer" {
  value = aws_opensearch_domain.es_server
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
