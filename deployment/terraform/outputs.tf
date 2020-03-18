output "instanceIPs" {
    value = "${aws_instance.app_server.*.public_ip}"
}

output "dbEndpoint" {
    value = "${aws_db_instance.db.endpoint}"
}