// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"

	"github.com/mattermost/mattermost/server/public/model"
)

func pickRate(config UserControllerConfiguration) (float64, error) {
	dist := config.RatesDistribution
	if len(dist) == 0 {
		return 1.0, nil
	}

	weights := make([]int, len(dist))
	for i := range dist {
		weights[i] = int(dist[i].Percentage * 100)
	}

	idx, err := control.SelectWeighted(weights)
	if err != nil {
		return -1, fmt.Errorf("loadtest: failed to select weight: %w", err)
	}

	return dist[idx].Rate, nil
}

// PromoteToAdmin promotes user to a sysadmin role
func PromoteToAdmin(admin, userForPromotion *userentity.UserEntity) error {
	isAdmin, err := admin.IsSysAdmin()
	if err != nil {
		return err
	}
	if !isAdmin {
		return errors.New("user is not an admin, cannot perform promoteAdmin")
	}

	isAdmin, err = userForPromotion.IsSysAdmin()
	if err != nil {
		return err
	}

	// User is already an admin, so early return
	if isAdmin {
		return nil
	}

	err = admin.UpdateUserRoles(userForPromotion.Store().Id(), fmt.Sprintf("%s %s", model.SystemUserRoleId, model.SystemAdminRoleId))
	if err != nil {
		return err
	}

	if err := userForPromotion.Login(); err != nil {
		return err
	}

	roleIds, err := userForPromotion.GetRolesByNames([]string{model.SystemUserRoleId, model.SystemAdminRoleId})
	if err != nil {
		return err
	}
	if len(roleIds) != 2 {
		return errors.New("user does not have the right roles updated")
	}
	if err := userForPromotion.Logout(); err != nil {
		return err
	}

	return nil
}

// Create CPA Fields
func CreateCustomAttributeFields(admin *userentity.UserEntity) error {
	fields := []model.PropertyField{
		model.PropertyField{
			Name: "TestCPAFieldOne",
			Type: "text",
		},
		model.PropertyField{
			Name: "TestCPAFieldOne",
			Type: "text",
		},
	}

	for _, field := range fields {
		_, err := admin.CreateCPAField(field)
		if err != nil {
			return err
		}
	}
	return nil
}

// nextPowerOf2 rounds its input value to the next power of 2 and returns it.
// courtesy of https://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2.
func nextPowerOf2(val int) int {
	val--
	val |= val >> 1
	val |= val >> 2
	val |= val >> 4
	val |= val >> 8
	val |= val >> 16
	val++
	return val
}

// RandomFutureTime returns a random Unix timestamp, in milliseconds, in the interval
// [now+deltaStart, now+deltaStart+maxUntil]
func RandomFutureTime(deltaStart, maxUntil time.Duration) int64 {
	now := time.Now()
	start := now.Add(deltaStart)
	start.Add(maxUntil)

	// Generate a random duration between 0 and maxUntil
	var randomDuration time.Duration
	if maxUntil > 0 {
		randomDuration = time.Duration(rand.Int63n(int64(maxUntil)))
	} else {
		randomDuration = time.Duration(0)
	}

	// Add the random duration to the start time
	randomTime := start.Add(randomDuration)

	return randomTime.UnixMilli()
}
