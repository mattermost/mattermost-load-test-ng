package terraform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPyroscopeConfigMarshal(t *testing.T) {
	data, err := NewPyroscopeConfig().marshal()
	require.NoError(t, err)
	require.Equal(t, `server:
    http_listen_port: 4040
limits:
    max_query_lookback: 30d
self_profiling:
    disable_push: true
`, string(data))
}

func TestAlloyConfigMarshal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  AlloyConfig
		output string
	}{
		{
			name: "empty",
			output: `logging {
	level = ""
}`,
		},
		{
			name:  "empty targets",
			input: NewAlloyConfig(nil, nil),
			output: `logging {
	level = "debug"
}

pyroscope.write "write_job" {
	endpoint {
		url = "http://localhost:4040"
	}
}

pyroscope.scrape "scrape_job" {
	targets    = []
	forward_to = [pyroscope.write.write_job.receiver]

	profiling_config {
		profile.process_cpu {
			enabled = true
		}

		profile.memory {
			enabled = true
		}

		profile.block {
			enabled = true
		}

		profile.goroutine {
			enabled = true
		}
	}
}`},
		{
			name:  "full",
			input: NewAlloyConfig([]string{"app-0:8067", "app-1:8067"}, []string{"agent-0:4000", "agent-1:4000"}),
			output: `logging {
	level = "debug"
}

pyroscope.write "write_job" {
	endpoint {
		url = "http://localhost:4040"
	}
}

pyroscope.scrape "scrape_job" {
	targets = [{
		__address__  = "app-0:8067",
		service_name = "app-0",
	}, {
		__address__  = "app-1:8067",
		service_name = "app-1",
	}, {
		__address__  = "agent-0:4000",
		service_name = "agent-0",
	}, {
		__address__  = "agent-1:4000",
		service_name = "agent-1",
	}]
	forward_to = [pyroscope.write.write_job.receiver]

	profiling_config {
		profile.process_cpu {
			enabled = true
		}

		profile.memory {
			enabled = true
		}

		profile.block {
			enabled = true
		}

		profile.goroutine {
			enabled = true
		}
	}
}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.input.marshal()
			require.NoError(t, err)
			require.Equal(t, tc.output, string(data))
		})
	}
}
