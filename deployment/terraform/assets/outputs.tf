output "instances" {
  value = "${aws_instance.app_server.*}"
}

output "dbCluster" {
  value = "${aws_rds_cluster.db_cluster}"
}

output "agents" {
  value = "${aws_instance.loadtest_agent.*}"
}

output "metricsServer" {
  value = "${aws_instance.metrics_server}"
}

output "proxy" {
  value = "${aws_instance.proxy_server}"
}

output "s3bucket" {
    value = "${aws_s3_bucket.s3bucket}"
}

output "s3Key" {
    value = "${aws_iam_access_key.s3key}"
}
