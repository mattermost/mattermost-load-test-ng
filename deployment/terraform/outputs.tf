output "instances" {
  value = "${aws_instance.app_server.*}"
}

output "dbEndpoint" {
  value = "${aws_db_instance.db.endpoint}"
}

output "metricsServer" {
  value = "${aws_instance.metrics_server}"
}
