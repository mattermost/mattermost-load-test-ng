package cluster

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockAgents(t *testing.T) []*agent.LoadAgent {
	cfg := agent.LoadAgentConfig{
		ApiURL: "api",
		Id:     "id",

		LoadTestConfig: loadtest.Config{
			ConnectionConfiguration: loadtest.ConnectionConfiguration{
				ServerURL:     "localhost:8065",
				WebSocketURL:  "localhost:4000",
				AdminEmail:    "user@example.com",
				AdminPassword: "str0ngPassword##",
			},
			UserControllerConfiguration: loadtest.UserControllerConfiguration{
				Type: "simple",
				Rate: 1.0,
			},
			UsersConfiguration: loadtest.UsersConfiguration{
				MaxActiveUsers:     8,
				InitialActiveUsers: 0,
			},
			InstanceConfiguration: loadtest.InstanceConfiguration{
				NumTeams: 1,
			},
			DeploymentConfiguration: loadtest.DeploymentConfiguration{
				DBInstanceEngine: "mysql",
			},
		},
	}
	agent1, err := agent.New(cfg)
	require.NoError(t, err)
	agent2, err := agent.New(cfg)
	require.NoError(t, err)
	agent3, err := agent.New(cfg)
	require.NoError(t, err)

	agents := []*agent.LoadAgent{
		agent1,
		agent2,
		agent3,
	}

	return agents
}

func TestAdditionDistribution(t *testing.T) {
	agents := createMockAgents(t)

	distribution, err := additionDistribution(agents[:1], 8)
	assert.NoError(t, err)
	assert.Equal(t, 8, distribution[0])

	agents[0].Status().NumUsers = 1
	agents[1].Status().NumUsers = 5

	distribution, err = additionDistribution(agents[:2], 8)
	assert.NoError(t, err)
	assert.Equal(t, 6, distribution[0])
	assert.Equal(t, 2, distribution[1])

	_, err = additionDistribution([]*agent.LoadAgent{}, 12)
	assert.Error(t, err)
}

func TestDeletionDistribution(t *testing.T) {
	agents := createMockAgents(t)

	distribution, err := deletionDistribution(agents[:1], 8)
	assert.NoError(t, err)
	assert.Equal(t, 0, distribution[0])

	agents[0].Status().NumUsers = 3
	agents[1].Status().NumUsers = 9

	distribution, err = deletionDistribution(agents, 8)
	assert.NoError(t, err)
	assert.Equal(t, 1, distribution[0])
	assert.Equal(t, 7, distribution[1])
	assert.Equal(t, 0, distribution[2])

	_, err = deletionDistribution([]*agent.LoadAgent{}, 12)
	assert.Error(t, err)
}
