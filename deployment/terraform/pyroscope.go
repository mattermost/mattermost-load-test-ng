package terraform

import (
	"strings"

	alloy "github.com/grafana/alloy/syntax"
	"github.com/grafana/alloy/syntax/token"
	"github.com/grafana/alloy/syntax/token/builder"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"
)

type PyroscopeTarget struct {
	Address     string `alloy:"__address__,attr"`
	ServiceName string `alloy:"service_name,attr"`
}

type PyroscopeProfilingConfig struct {
	ProcessCPU struct {
		Enabled bool `alloy:"enabled,attr"`
	} `alloy:"profile.process_cpu,block"`
	Memory struct {
		Enabled bool `alloy:"enabled,attr"`
	} `alloy:"profile.memory,block"`
	Block struct {
		Enabled bool `alloy:"enabled,attr"`
	} `alloy:"profile.block,block"`
	Goroutine struct {
		Enabled bool `alloy:"enabled,attr"`
	} `alloy:"profile.goroutine,block"`
}

type AlloyLiteralValue string

func (s AlloyLiteralValue) AlloyTokenize() []builder.Token {
	return []builder.Token{{Tok: token.LITERAL, Lit: string(s)}}
}

type AlloyConfig struct {
	Logging struct {
		Level string `alloy:"level,attr"`
	} `alloy:"logging,block"`

	PyroscopeWrite struct {
		Label    string `alloy:",label"`
		Endpoint struct {
			URL string `alloy:"url,attr"`
		} `alloy:"endpoint,block"`
	} `alloy:"pyroscope.write,block,optional"`

	PyroscopeScrape struct {
		Label           string                   `alloy:",label"`
		Targets         []PyroscopeTarget        `alloy:"targets,attr"`
		ForwardTo       []AlloyLiteralValue      `alloy:"forward_to,attr"`
		ProfilingConfig PyroscopeProfilingConfig `alloy:"profiling_config,block"`
	} `alloy:"pyroscope.scrape,block,optional"`
}

func (c AlloyConfig) marshal() ([]byte, error) {
	return alloy.Marshal(c)
}

func NewAlloyConfig(mmTargets, ltTargets []string) AlloyConfig {
	var c AlloyConfig

	c.Logging.Level = "debug"

	c.PyroscopeWrite.Label = "write_job"
	c.PyroscopeWrite.Endpoint.URL = "http://localhost:4040"

	c.PyroscopeScrape.Label = "scrape_job"
	c.PyroscopeScrape.ForwardTo = []AlloyLiteralValue{"pyroscope.write.write_job.receiver"}
	c.PyroscopeScrape.ProfilingConfig.ProcessCPU.Enabled = true
	c.PyroscopeScrape.ProfilingConfig.Memory.Enabled = true
	c.PyroscopeScrape.ProfilingConfig.Block.Enabled = true
	c.PyroscopeScrape.ProfilingConfig.Goroutine.Enabled = true

	for _, target := range append(mmTargets, ltTargets...) {
		serviceName, _, _ := strings.Cut(target, ":")
		c.PyroscopeScrape.Targets = append(c.PyroscopeScrape.Targets, PyroscopeTarget{
			Address:     target,
			ServiceName: serviceName,
		})
	}

	return c
}

type PyroscopeConfig struct {
	Server struct {
		HTTPListenPort int `yaml:"http_listen_port"`
	} `yaml:"server"`
	Limits struct {
		MaxQueryLookback model.Duration `yaml:"max_query_lookback"`
	} `yaml:"limits"`
	SelfProfiling struct {
		DisablePush bool `yaml:"disable_push"`
	} `yaml:"self_profiling"`
}

func (c PyroscopeConfig) marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

func NewPyroscopeConfig() PyroscopeConfig {
	var c PyroscopeConfig
	c.Server.HTTPListenPort = 4040
	c.SelfProfiling.DisablePush = true
	c.Limits.MaxQueryLookback, _ = model.ParseDuration("30d")
	return c
}
