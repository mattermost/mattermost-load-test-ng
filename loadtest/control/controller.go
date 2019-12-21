// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

type UserController interface {
	Run()
	SetRate(rate float64) error
	Stop()
}
