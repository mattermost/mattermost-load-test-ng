// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package accesscontrol sets up Attribute-Based Access Control (ABAC) on the
// target instance for a load-test: user attributes, session attributes and
// permission policies governing channel read and write access. It also
// provides the per-user values the simulated clients send and receive.
package accesscontrol

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	ActionChannelRead  = "channel_read_access"
	ActionChannelWrite = "channel_write_access"

	OperatorAnd = "and"
	OperatorOr  = "or"

	AttributeTypeSelect = "select"
	AttributeTypeText   = "text"

	RoleSystemUser  = "system_user"
	RoleSystemAdmin = "system_admin"
	RoleSystemGuest = "system_guest"

	// The server only evaluates the first 10 permission policies it finds.
	maxPermissionPolicies = 10
	// The server allows at most 20 user attribute fields.
	maxUserAttributes = 20
	// Policy rule names are limited to 128 characters by the server.
	maxPolicyNameLength = 128
)

var attributeNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// clientSessionAttributes are the session attributes the server accepts from
// clients through the X-MM-Session-Attributes header, mapped to their allowed
// values (nil means free text). Attributes that are computed by the server
// from the request (ip_address, user_agent_*) are not part of this list.
//
// Note that the server also filters them by the client platform inferred from
// the User-Agent: jailbreak_detected and client_device_id are only accepted
// from mobile clients, hardware_id and client_fqdn only from desktop clients.
var clientSessionAttributes = map[string][]string{
	"client_ip_address":      nil,
	"network_interface_type": {"wifi", "ethernet", "cellular", "vpn", "bluetooth", "other"},
	"vpn_active":             {"true", "false"},
	"ssid":                   nil,
	"mdm_enrolled":           {"true", "false"},
	"jailbreak_detected":     {"true", "false"},
	"os_platform":            {"macos", "windows", "linux", "ios", "android"},
	"os_version":             nil,
	"client_version":         nil,
	"client_device_id":       nil,
	"hardware_id":            nil,
	"server_fqdn":            nil,
	"client_fqdn":            nil,
}

// AttributeValue is a possible value of an attribute along with the
// probability of it being assigned.
type AttributeValue struct {
	Value string `validate:"notempty"`
	// The probability of a user (or session) being assigned this value.
	// Weights of all the values of an attribute must sum to 1.
	Weight float64 `validate:"range:[0,1]"`
}

// UserAttribute describes a user attribute (custom profile attribute) to be
// created and randomly assigned to users.
type UserAttribute struct {
	// The name of the attribute, as referenced in policy expressions
	// (user.attributes.<Name>).
	Name string `validate:"notempty"`
	// The type of the attribute. Either "select" or "text".
	// Defaults to "select".
	Type string
	// The values users get assigned. For select attributes these become the
	// options of the field.
	Values []AttributeValue
}

// SessionAttribute describes a client-provided session attribute to be
// enabled on the server and randomly assigned to the simulated clients, which
// send it on every request.
type SessionAttribute struct {
	// The name of the attribute, as referenced in policy expressions
	// (user.session.<Name>).
	Name string `validate:"notempty"`
	// If greater than zero, overrides the time (in seconds) the server
	// considers the attribute fresh after the last request carrying it.
	TTLSeconds int `default:"0" validate:"range:[0,]"`
	// If greater than zero, overrides the extra time (in seconds) the server
	// keeps a stale attribute before discarding it.
	GracePeriodSeconds int `default:"0" validate:"range:[0,]"`
	// The values the simulated clients get assigned.
	Values []AttributeValue
}

// Policy describes a permission policy that governs an action on every public
// and private channel. It combines one user attribute condition and one
// session attribute condition:
//
//	user.attributes.<UserAttribute> in [<UserAttributeValues>]
//	  && (or ||)
//	user.session.<SessionAttribute> in [<SessionAttributeValues>]
type Policy struct {
	// The unique name of the policy.
	Name string `validate:"notempty"`
	// The action governed by the policy. Either "channel_read_access" or
	// "channel_write_access".
	Action string `validate:"oneof:{channel_read_access,channel_write_access}"`
	// The system role the policy applies to. Defaults to "system_user".
	// System admins without a dedicated policy fall back to the system_user
	// policies.
	Role string
	// The user attribute to check, and the values that grant access.
	UserAttribute       string `validate:"notempty"`
	UserAttributeValues []string
	// How the two conditions are combined. Either "and" or "or".
	// Defaults to "and".
	Operator string
	// The session attribute to check, and the values that grant access.
	SessionAttribute       string `validate:"notempty"`
	SessionAttributeValues []string
}

// Config holds the ABAC setup of a load-test.
type Config struct {
	// Enable sets up the attributes and policies on the target instance and
	// makes the simulated users send their session attributes.
	// Requires an Enterprise Advanced license and the target instance to run
	// with the SessionAttributes feature flag and
	// AccessControlSettings.EnableAttributeBasedAccessControl enabled.
	Enable bool `default:"false"`
	// The User-Agent the simulated users send when Enable is true. The
	// server only accepts client session attributes from desktop and mobile
	// apps, which it recognizes by their User-Agent.
	UserAgent string `default:"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Mattermost/5.13.0 Chrome/136.0.7103.149 Electron/36.4.0 Safari/537.36"`
	// The user attributes to create and assign.
	UserAttributes []UserAttribute
	// The client-provided session attributes to enable and send.
	SessionAttributes []SessionAttribute
	// The permission policies to create.
	Policies []Policy
}

// IsValid reports whether a given Config is valid or not.
func (c Config) IsValid() error {
	if !c.Enable {
		return nil
	}

	if c.UserAgent == "" {
		return errors.New("accesscontrol: UserAgent cannot be empty")
	}

	if len(c.UserAttributes) > maxUserAttributes {
		return fmt.Errorf("accesscontrol: at most %d user attributes are supported", maxUserAttributes)
	}
	userAttrs := make(map[string]UserAttribute, len(c.UserAttributes))
	for _, attr := range c.UserAttributes {
		if !attributeNameRE.MatchString(attr.Name) {
			return fmt.Errorf("accesscontrol: invalid user attribute name %q", attr.Name)
		}
		if _, ok := userAttrs[attr.Name]; ok {
			return fmt.Errorf("accesscontrol: duplicate user attribute %q", attr.Name)
		}
		if attr.Type != "" && attr.Type != AttributeTypeSelect && attr.Type != AttributeTypeText {
			return fmt.Errorf("accesscontrol: invalid type %q for user attribute %q", attr.Type, attr.Name)
		}
		if err := validateValues(attr.Values, nil); err != nil {
			return fmt.Errorf("accesscontrol: user attribute %q: %w", attr.Name, err)
		}
		userAttrs[attr.Name] = attr
	}

	sessionAttrs := make(map[string]SessionAttribute, len(c.SessionAttributes))
	for _, attr := range c.SessionAttributes {
		allowed, ok := clientSessionAttributes[attr.Name]
		if !ok {
			return fmt.Errorf("accesscontrol: %q is not a session attribute provided by clients", attr.Name)
		}
		if _, ok := sessionAttrs[attr.Name]; ok {
			return fmt.Errorf("accesscontrol: duplicate session attribute %q", attr.Name)
		}
		if err := validateValues(attr.Values, allowed); err != nil {
			return fmt.Errorf("accesscontrol: session attribute %q: %w", attr.Name, err)
		}
		sessionAttrs[attr.Name] = attr
	}

	if len(c.Policies) == 0 {
		return errors.New("accesscontrol: at least one policy is required")
	}
	if len(c.Policies) > maxPermissionPolicies {
		return fmt.Errorf("accesscontrol: at most %d policies are supported, since the server only evaluates the first %d permission policies", maxPermissionPolicies, maxPermissionPolicies)
	}
	policyNames := make(map[string]bool, len(c.Policies))
	for _, p := range c.Policies {
		if len(p.Name) > maxPolicyNameLength {
			return fmt.Errorf("accesscontrol: policy name %q is longer than %d characters", p.Name, maxPolicyNameLength)
		}
		if policyNames[p.Name] {
			return fmt.Errorf("accesscontrol: duplicate policy %q", p.Name)
		}
		policyNames[p.Name] = true
		if p.Role != "" && p.Role != RoleSystemUser && p.Role != RoleSystemAdmin && p.Role != RoleSystemGuest {
			return fmt.Errorf("accesscontrol: invalid role %q for policy %q", p.Role, p.Name)
		}
		if p.Operator != "" && p.Operator != OperatorAnd && p.Operator != OperatorOr {
			return fmt.Errorf("accesscontrol: invalid operator %q for policy %q", p.Operator, p.Name)
		}
		userAttr, ok := userAttrs[p.UserAttribute]
		if !ok {
			return fmt.Errorf("accesscontrol: policy %q references undefined user attribute %q", p.Name, p.UserAttribute)
		}
		if err := validatePolicyValues(p.UserAttributeValues, userAttr.Values); err != nil {
			return fmt.Errorf("accesscontrol: policy %q: user attribute %q: %w", p.Name, p.UserAttribute, err)
		}
		sessionAttr, ok := sessionAttrs[p.SessionAttribute]
		if !ok {
			return fmt.Errorf("accesscontrol: policy %q references undefined session attribute %q", p.Name, p.SessionAttribute)
		}
		if err := validatePolicyValues(p.SessionAttributeValues, sessionAttr.Values); err != nil {
			return fmt.Errorf("accesscontrol: policy %q: session attribute %q: %w", p.Name, p.SessionAttribute, err)
		}
	}

	return nil
}

func validateValues(values []AttributeValue, allowed []string) error {
	if len(values) == 0 {
		return errors.New("at least one value is required")
	}
	var sum float64
	seen := make(map[string]bool, len(values))
	for _, v := range values {
		if seen[v.Value] {
			return fmt.Errorf("duplicate value %q", v.Value)
		}
		seen[v.Value] = true
		if allowed != nil && !slices.Contains(allowed, v.Value) {
			return fmt.Errorf("invalid value %q, allowed values are %v", v.Value, allowed)
		}
		sum += v.Weight
	}
	if math.Abs(sum-1) > 1e-6 {
		return fmt.Errorf("weights should sum to 1, got %v", sum)
	}
	return nil
}

func validatePolicyValues(policyValues []string, values []AttributeValue) error {
	if len(policyValues) == 0 {
		return errors.New("at least one value is required")
	}
	for _, pv := range policyValues {
		if !slices.ContainsFunc(values, func(v AttributeValue) bool { return v.Value == pv }) {
			return fmt.Errorf("value %q is not one of the attribute values", pv)
		}
	}
	return nil
}

func (p Policy) role() string {
	if p.Role == "" {
		return RoleSystemUser
	}
	return p.Role
}

func (a UserAttribute) attrType() string {
	if a.Type == "" {
		return AttributeTypeSelect
	}
	return a.Type
}

// Expression returns the CEL expression of the policy.
func (p Policy) Expression() string {
	op := "&&"
	if p.Operator == OperatorOr {
		op = "||"
	}
	return fmt.Sprintf("user.attributes.%s in %s %s user.session.%s in %s",
		p.UserAttribute, celList(p.UserAttributeValues), op, p.SessionAttribute, celList(p.SessionAttributeValues))
}

func celList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// permits reports whether the policy grants access given the assigned user
// and session attribute values.
func (p Policy) permits(userValues, sessionValues map[string]string) bool {
	userMatch := slices.Contains(p.UserAttributeValues, userValues[p.UserAttribute])
	sessionMatch := slices.Contains(p.SessionAttributeValues, sessionValues[p.SessionAttribute])
	if p.Operator == OperatorOr {
		return userMatch || sessionMatch
	}
	return userMatch && sessionMatch
}
