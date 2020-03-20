output "instances" {
  value = "${aws_instance.app_server.*}"
}

output "dbCluster" {
  value = "${aws_rds_cluster.db_cluster}"
}

output "metricsServer" {
  value = "${aws_instance.metrics_server}"
}