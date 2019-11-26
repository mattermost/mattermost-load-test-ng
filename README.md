# Mattermost Load Test 

Mattermost Load Test provides a set of tools written in [Go](https://golang.org/) for profiling [Mattermost](https://github.com/mattermost/mattermost-server) under heavy load simulating real-world usage of the Mattermost Enterprise Edition E20 at scale.

In setting up this system you will: 

- Deploy Mattermost in a production configuration, potentially in high availability mode
- Optimize the performance of your Mattermost deployment 
- Deploy a Mattermost Load Test server to apply simulated load to your production deployment 
- Log in to your Mattermost deployment to observe it under load

If you have questions about configuration, please contact your Customer Success Manager. An overview of support available to E20 customers is available at https://about.mattermost.com/support/

## Installation

Install the binaries:
```
go get github.com/mattermost/mattermost-load-test/cmd/ltops
go get github.com/mattermost/mattermost-load-test/cmd/loadtest
go get github.com/mattermost/mattermost-load-test/cmd/ltparse
```

Run `ltops help` to verify the installation and get started with the available commands.

## Profiling Strategies

Various profiling strategies are currently supported:
* [AWS cluster using Terraform](docs/terraform.md) (recommended)
* [Kubernetes cluster](docs/kubernetes.md)
* [Manual loadtesting against an existing cluster](docs/manual.md)

The best way to profile the mattermost-server is to set up an [AWS cluster using Terraform](docs/terraform.md) using the `ltops` tool. Use this setup to qualify the performance of a given Mattermost release, or measure the effect of an experimental change to the [mattermost-server](https://github.com/mattermost/mattermost-server). Note that while other cloud providers are on the roadmap, only AWS is supported at present.

Feel free to experiment with profiling a Mattermost [Kubernetes cluster](docs/kubernetes.md) using the `ltops` tool, but recognize that this is still in beta. There may also be some tooling differences between Kubernetes and the more stable Terraform setup.

The `loadtest` tool may be run [manually against an existing cluster](docs/manual.md), regardless of how that cluster is deployed. Note that care is required to tune an arbitrary cluster to perform well under load. This method of profiling is also suitable for basic `localhost` profiling, especially when developing against mattermost-load-test itself.

## Development

Follow the [Mattermost developer setup instructions](https://developers.mattermost.com/contribute/server/developer-setup/), then clone the repository and build for yourself:

```
go get github.com/mattermost/mattermost-load-test
cd $(go env GOPATH)/src/mattermost/mattermost-load-test
make install
```
