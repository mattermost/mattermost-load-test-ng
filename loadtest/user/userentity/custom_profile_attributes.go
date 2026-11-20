package userentity

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// CreateCPAField creates a new custom profile attribute field.
func (ue *UserEntity) CreateCPAField(field *model.PropertyField) (*model.PropertyField, error) {
	newField, _, err := ue.client.CreateCPAField(context.Background(), field)
	if err != nil {
		return nil, err
	}
	return newField, nil
}

// GetCPAFields retrieves all the cpa fields available.
func (ue *UserEntity) GetCPAFields() error {
	fields, _, err := ue.client.ListCPAFields(context.Background())
	if err != nil {
		return err
	}
	return ue.store.SetCPAFields(fields)
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

// PatchCPAValues patches (or creates) a given user's custom profile attributes.
func (ue *UserEntity) PatchCPAValues(values map[string]json.RawMessage) error {
	_, _, err := ue.client.PatchCPAValues(context.Background(), values)
	return err
}

// PatchCPAField patches the given custom profile attribute field.
func (ue *UserEntity) PatchCPAField(fieldID string, patch *model.PropertyFieldPatch) (*model.PropertyField, error) {
	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("patch_attribute_field", time.Since(start).Seconds())
	}(time.Now())

	field, _, err := ue.client.PatchCPAField(context.Background(), fieldID, patch)
	if err != nil {
		return nil, err
	}
	return field, nil
}

// PatchCPAValuesForUser patches (or creates) the custom profile attributes of
// the given user.
func (ue *UserEntity) PatchCPAValuesForUser(userID string, values map[string]json.RawMessage) error {
	defer func(start time.Time) {
		ue.observeScopedConfigRequestTime("patch_user_attributes", time.Since(start).Seconds())
	}(time.Now())

	_, _, err := ue.client.PatchCPAValuesForUser(context.Background(), userID, values)
	return err
}
