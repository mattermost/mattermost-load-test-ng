// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	sessionAttributesGroupName  = "session_attributes"
	sessionAttributesObjectType = "session"
	sessionAttributesTargetType = "system"
)

func (ue *UserEntity) doScopedConfigRequest(method, path string, body, out any) error {
	var data io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		data = bytes.NewReader(buf)
	}

	req, err := ue.prepareRequest(method, ue.client.APIURL+path, data, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}

	resp, err := ue.client.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s %s failed with status %d: %w", method, path, resp.StatusCode, model.AppErrorFromJSON(resp.Body))
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// GetClientConfigOverrides fetches and stores the client config values
// overridden for the logged in user by system setting policies.
func (ue *UserEntity) GetClientConfigOverrides() error {
	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("get_client_config_overrides", time.Since(start).Seconds())
	}(time.Now())

	var overrides map[string]string
	if err := ue.doScopedConfigRequest(http.MethodGet, "/config/client/overrides", nil, &overrides); err != nil {
		return err
	}

	ue.store.SetClientConfigOverrides(overrides)

	return nil
}

// SearchSystemSettingPolicies returns all the system setting policies.
func (ue *UserEntity) SearchSystemSettingPolicies() ([]*user.SystemSettingPolicy, error) {
	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("search_policies", time.Since(start).Seconds())
	}(time.Now())

	var res struct {
		Policies []*user.SystemSettingPolicy `json:"policies"`
	}
	search := map[string]any{
		"type":  user.SystemSettingPolicyType,
		"limit": 100,
	}
	if err := ue.doScopedConfigRequest(http.MethodPost, "/access_control_policies/search", search, &res); err != nil {
		return nil, err
	}

	return res.Policies, nil
}

// GetSystemSettingPolicy returns the given system setting policy. Unlike
// search results, its expressions reference attributes by name.
func (ue *UserEntity) GetSystemSettingPolicy(policyID string) (*user.SystemSettingPolicy, error) {
	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("get_policy", time.Since(start).Seconds())
	}(time.Now())

	var policy user.SystemSettingPolicy
	if err := ue.doScopedConfigRequest(http.MethodGet, "/access_control_policies/"+policyID, nil, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// SaveSystemSettingPolicy creates or updates the given system setting policy.
func (ue *UserEntity) SaveSystemSettingPolicy(policy *user.SystemSettingPolicy) (*user.SystemSettingPolicy, error) {
	if policy == nil {
		return nil, errors.New("policy should not be nil")
	}

	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("save_policy", time.Since(start).Seconds())
	}(time.Now())

	var saved user.SystemSettingPolicy
	if err := ue.doScopedConfigRequest(http.MethodPut, "/access_control_policies", policy, &saved); err != nil {
		return nil, err
	}

	return &saved, nil
}

// DeleteSystemSettingPolicy deletes the given system setting policy.
func (ue *UserEntity) DeleteSystemSettingPolicy(policyID string) error {
	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("delete_policy", time.Since(start).Seconds())
	}(time.Now())

	_, err := ue.client.DeleteAccessControlPolicy(context.Background(), policyID)
	return err
}

// GetSessionAttributeFields returns the session attribute fields.
func (ue *UserEntity) GetSessionAttributeFields() ([]*model.PropertyField, error) {
	fields, _, err := ue.client.GetPropertyFields(context.Background(), sessionAttributesGroupName, sessionAttributesObjectType, model.PropertyFieldSearch{
		TargetType: sessionAttributesTargetType,
		PerPage:    100,
	})
	if err != nil {
		return nil, err
	}

	return fields, nil
}

// PatchSessionAttributeField patches the given session attribute field.
func (ue *UserEntity) PatchSessionAttributeField(fieldID string, patch *model.PropertyFieldPatch) (*model.PropertyField, error) {
	field, _, err := ue.client.PatchPropertyField(context.Background(), sessionAttributesGroupName, sessionAttributesObjectType, fieldID, patch)
	if err != nil {
		return nil, err
	}

	return field, nil
}

func (ue *UserEntity) handleSystemSettingPolicyUpdatedEvent(ev *model.WebSocketEvent) error {
	ue.incSystemSettingPolicyUpdatedWSEvents()

	data, ok := ev.GetData()["overrides"].(map[string]any)
	if !ok {
		return fmt.Errorf("type of the overrides data should be a map, but it is %T", ev.GetData()["overrides"])
	}

	overrides := make(map[string]string, len(data))
	for k, v := range data {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("type of the override value for %q should be a string, but it is %T", k, v)
		}
		overrides[k] = s
	}

	ue.store.SetClientConfigOverrides(overrides)

	return nil
}
