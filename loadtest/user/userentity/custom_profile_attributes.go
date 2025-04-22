package userentity

import (
	"context"
	"encoding/json"

	"github.com/mattermost/mattermost/server/public/model"
)

// CreateCPAField creates a new custom profile attribute field.
func (ue *UserEntity) CreateCPAField(field model.PropertyField) (*model.PropertyField, error) {
	new_field, _, err := ue.client.CreateCPAField(context.Background(), &field)
	if err != nil {
		return nil, err
	}
	return new_field, nil
}

// GetCPAFeidlds retrieves all the cpa field values available.
func (ue *UserEntity) GetCPAFields() error {
	fields, _, err := ue.client.ListCPAFields(context.Background())
	if err != nil {
		return err
	}
	err = ue.store.SetCPAFields(fields)
	if err != nil {
		return err
	}

	return nil
}

// GetCPAValues returns all the custom profile attributes for a user
func (ue *UserEntity) GetCPAValues(userId string) (map[string]json.RawMessage, error) {
	values, _, err := ue.client.ListCPAValues(context.Background(), userId)
	if err != nil {
		return nil, err
	}

	err = ue.store.SetCPAValues(userId, values)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// PatchUserCPA patches (or creates) a given users custom profile attributes.
func (ue *UserEntity) PatchCPAValues(userId string, values map[string]json.RawMessage) error {
	_, _, err := ue.client.PatchCPAValues(context.Background(), values)
	if err != nil {
		return err
	}
	return nil
}
