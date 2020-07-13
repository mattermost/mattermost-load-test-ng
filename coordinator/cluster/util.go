package cluster

import (
	"errors"
	"fmt"
	"sort"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/agent"
)

type sortableAgent struct {
	index int
	users int
}

func getUsersAmounts(agents []*client.Agent) ([]int, error) {
	amounts := make([]int, len(agents))
	for i, a := range agents {
		// TODO: possibly optimize this either by running goroutines to make the
		// requests concurrently or by caching agents' statuses.
		status, err := a.Status()
		if err != nil {
			return nil, fmt.Errorf("failed to get status for agent: %w", err)
		}
		amounts[i] = int(status.NumUsers)
	}
	return amounts, nil
}

// gives numbers to distribute users evenly
func populateSortableAgents(amounts []int) ([]sortableAgent, error) {
	if len(amounts) == 0 {
		return nil, errors.New("input slice length must be greater than 0")
	}
	sortableAgents := make([]sortableAgent, len(amounts))
	for i, a := range amounts {
		sortableAgents[i].index = i
		sortableAgents[i].users = a
	}
	return sortableAgents, nil
}

func additionDistribution(amounts []int, n int) (map[int]int, error) {
	sortableAgents, err := populateSortableAgents(amounts)
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

func deletionDistribution(amounts []int, n int) (map[int]int, error) {
	sortableAgents, err := populateSortableAgents(amounts)
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
