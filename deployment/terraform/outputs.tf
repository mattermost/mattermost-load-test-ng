output "instanceIPs" {
    value = "${aws_instance.app_server.*.public_ip}"
}

