module github.com/mattermost/mattermost-load-test-ng

go 1.12

require (
	github.com/gorilla/mux v1.7.3
	github.com/mattermost/mattermost-server v5.11.1+incompatible
	github.com/mattermost/mattermost-server/v5 v5.18.0-rc1
	github.com/onsi/ginkgo v1.10.2 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/satori/go.uuid v0.0.0-20180103174451-36e9d2ebbde5
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190515063710-7b18d6600f6b

replace github.com/codegangsta/cli v1.22.1 => github.com/urfave/cli v1.22.1
