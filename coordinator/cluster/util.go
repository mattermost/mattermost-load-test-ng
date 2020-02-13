package cluster

import (
	"errors"
	"sort"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
)

type sortableAgent struct {
	order int
	Users int
}

// gives numbers to distribute users evenly
func populateSortableAgents(agents []*agent.LoadAgent) ([]sortableAgent, error) {
	sortedAgents := make([]sortableAgent, 0)
	if len(agents) == 0 {
		return nil, errors.New("input slice length must be greater than 0")
	}
	for i, a := range agents {
		sortedAgents = append(sortedAgents, sortableAgent{
			order: i,
			Users: a.Status().NumUsers,
		})
	}
	return sortedAgents, nil
}

func additionDistribution(agents []*agent.LoadAgent, n int) (map[int]int, error) {
	sortedAgents, err := populateSortableAgents(agents)
	if err != nil {
		return nil, err
	}
	distMap := make(map[int]int)
	for i := 0; i < n; i++ {
		sort.Slice(sortedAgents, func(i, j int) bool {
			return sortedAgents[i].Users < sortedAgents[j].Users
		})
		sortedAgents[0].Users++
		distMap[sortedAgents[0].order] = distMap[sortedAgents[0].order] + 1
	}
	return distMap, nil
}

func deletionDistribution(agents []*agent.LoadAgent, n int) (map[int]int, error) {
	sortedAgents, err := populateSortableAgents(agents)
	if err != nil {
		return nil, err
	}
	distMap := make(map[int]int)
	for i := 0; i < n; i++ {
		sort.Slice(sortedAgents, func(i, j int) bool {
			return sortedAgents[i].Users > sortedAgents[j].Users
		})
		if sortedAgents[0].Users > 0 {
			sortedAgents[0].Users--
			distMap[sortedAgents[0].order] = distMap[sortedAgents[0].order] + 1
		}
	}
	return distMap, nil
}
