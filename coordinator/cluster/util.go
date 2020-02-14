package cluster

import (
	"errors"
	"sort"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
)

type sortableAgent struct {
	index int
	users int
}

// gives numbers to distribute users evenly
func populateSortableAgents(agents []*agent.LoadAgent) ([]sortableAgent, error) {
	if len(agents) == 0 {
		return nil, errors.New("input slice length must be greater than 0")
	}
	sortableAgents := make([]sortableAgent, len(agents))
	for i, a := range agents {
		sortableAgents[i] = sortableAgent{
			index: i,
			users: a.Status().NumUsers,
		}
	}
	return sortableAgents, nil
}

func additionDistribution(agents []*agent.LoadAgent, n int) (map[int]int, error) {
	sortableAgents, err := populateSortableAgents(agents)
	if err != nil {
		return nil, err
	}
	distMap := make(map[int]int)
	for i := 0; i < n; i++ {
		sort.Slice(sortableAgents, func(i, j int) bool {
			return sortableAgents[i].users < sortableAgents[j].users
		})
		sortableAgents[0].users++
		distMap[sortableAgents[0].index] = distMap[sortableAgents[0].index] + 1
	}
	return distMap, nil
}

func deletionDistribution(agents []*agent.LoadAgent, n int) (map[int]int, error) {
	sortableAgents, err := populateSortableAgents(agents)
	if err != nil {
		return nil, err
	}
	distMap := make(map[int]int)
	for i := 0; i < n; i++ {
		sort.Slice(sortableAgents, func(i, j int) bool {
			return sortableAgents[i].users > sortableAgents[j].users
		})
		if sortableAgents[0].users > 0 {
			sortableAgents[0].users--
			distMap[sortableAgents[0].index] = distMap[sortableAgents[0].index] + 1
		}
	}
	return distMap, nil
}
