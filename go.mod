module github.com/mattermost/mattermost-load-test-ng

go 1.12

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/fatih/color v1.9.0
	github.com/gavv/httpexpect v2.0.0+incompatible
	github.com/gocolly/colly/v2 v2.0.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/mattermost/ldap v3.0.4+incompatible // indirect
	github.com/mattermost/mattermost-server/v5 v5.3.2-0.20201216152535-675fab8ab63a
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.14.0
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/valyala/fasthttp v1.7.1 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190515063710-7b18d6600f6b

replace github.com/codegangsta/cli v1.22.1 => github.com/urfave/cli v1.22.1
