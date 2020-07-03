package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdditionDistribution(t *testing.T) {
	amounts := []int{0, 0, 0}

	distribution, err := additionDistribution(amounts[:1], 8)
	assert.NoError(t, err)
	assert.Equal(t, 8, distribution[0])

	amounts[0] = 1
	amounts[1] = 5

	distribution, err = additionDistribution(amounts[:2], 8)
	assert.NoError(t, err)
	assert.Equal(t, 6, distribution[0])
	assert.Equal(t, 2, distribution[1])

	_, err = additionDistribution([]int{}, 12)
	assert.Error(t, err)
}

func TestDeletionDistribution(t *testing.T) {
	amounts := []int{0, 0, 0}

	distribution, err := deletionDistribution(amounts[:1], 8)
	assert.NoError(t, err)
	assert.Equal(t, 0, distribution[0])

	amounts[0] = 3
	amounts[1] = 9

	distribution, err = deletionDistribution(amounts, 8)
	assert.NoError(t, err)
	assert.Equal(t, 1, distribution[0])
	assert.Equal(t, 7, distribution[1])
	assert.Equal(t, 0, distribution[2])

	_, err = deletionDistribution([]int{}, 12)
	assert.Error(t, err)
}
