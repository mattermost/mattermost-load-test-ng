module github.com/mattermost/mattermost-load-test

go 1.12

require (
	github.com/go-redis/redis v6.15.5+incompatible // indirect
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/lib/pq v1.2.0 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mattermost/go-i18n v1.11.0 // indirect
	github.com/mattermost/mattermost-server v0.0.0-20190913222010-f4f7fd0829d7
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/onsi/ginkgo v1.10.2 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.4.0 // indirect
	go.uber.org/multierr v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20191002192127-34f69633bfdc // indirect
	golang.org/x/net v0.0.0-20191002035440-2ec189313ef0 // indirect
	golang.org/x/sys v0.0.0-20191002091554-b397fe3ad8ed // indirect
	google.golang.org/appengine v1.6.4 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190515063710-7b18d6600f6b

replace github.com/codegangsta/cli v1.22.1 => github.com/urfave/cli v1.22.1

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190930215403-16217165b5de
