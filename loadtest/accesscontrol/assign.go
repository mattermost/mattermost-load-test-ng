// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package accesscontrol

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math/rand"
	"strings"
)

const (
	// HeaderSessionAttributes is the header through which clients send their
	// session attributes, as base64-encoded JSON.
	HeaderSessionAttributes = "X-MM-Session-Attributes"
	headerUserAgent         = "User-Agent"

	// Above this number of attribute value combinations, the expected permit
	// rates are estimated through sampling instead of computed exactly.
	maxExactCombinations = 1 << 20
	numRateSamples       = 200000
)

// pickValue deterministically picks one of the values based on their weights
// and the given key, so that a user always gets the same value, regardless of
// the agent simulating it or how many times it logs in.
func pickValue(key string, values []AttributeValue) string {
	sum := sha256.Sum256([]byte(key))
	// Use the top 53 bits to get a uniformly distributed float in [0, 1).
	r := float64(binary.BigEndian.Uint64(sum[:8])>>11) / (1 << 53)
	return pickValueAt(r, values)
}

func pickValueAt(r float64, values []AttributeValue) string {
	var cum float64
	for _, v := range values {
		cum += v.Weight
		if r < cum {
			return v.Value
		}
	}
	return values[len(values)-1].Value
}

func userKey(email string) string {
	return strings.ToLower(email)
}

// UserAttributeValues returns the user attribute values (by attribute name)
// assigned to the user with the given email.
func (c Config) UserAttributeValues(email string) map[string]string {
	values := make(map[string]string, len(c.UserAttributes))
	for _, attr := range c.UserAttributes {
		values[attr.Name] = pickValue("user:"+userKey(email)+":"+attr.Name, attr.Values)
	}
	return values
}

// SessionAttributeValues returns the session attribute values (by attribute
// name) the user with the given email sends.
func (c Config) SessionAttributeValues(email string) map[string]string {
	values := make(map[string]string, len(c.SessionAttributes))
	for _, attr := range c.SessionAttributes {
		values[attr.Name] = pickValue("session:"+userKey(email)+":"+attr.Name, attr.Values)
	}
	return values
}

// Headers returns the HTTP headers the user with the given email sends on
// every request to report its session attributes, or nil if the setup is
// disabled.
func (c Config) Headers(email string) map[string]string {
	if !c.Enable {
		return nil
	}
	headers := map[string]string{
		headerUserAgent: c.UserAgent,
	}
	if len(c.SessionAttributes) > 0 {
		// Marshalling a map[string]string cannot fail.
		data, _ := json.Marshal(c.SessionAttributeValues(email))
		headers[HeaderSessionAttributes] = base64.StdEncoding.EncodeToString(data)
	}
	return headers
}

// PermitRates holds the expected share of users that are granted access.
type PermitRates struct {
	// The share of users passing all the channel_read_access policies.
	Read float64
	// The share of users passing all the channel_read_access and
	// channel_write_access policies, since the server requires read access
	// to grant write access.
	Write float64
}

// ExpectedPermitRates computes the expected share of users that are granted
// channel read and write access by the configured policies, given the weights
// of the attribute values.
func (c Config) ExpectedPermitRates() PermitRates {
	var readPolicies, writePolicies []Policy
	for _, p := range c.Policies {
		// Only system_user policies apply to the regular simulated users.
		if p.role() != RoleSystemUser {
			continue
		}
		switch p.Action {
		case ActionChannelRead:
			readPolicies = append(readPolicies, p)
		case ActionChannelWrite:
			writePolicies = append(writePolicies, p)
		}
	}

	var rates PermitRates
	c.forEachCombination(func(prob float64, userValues, sessionValues map[string]string) {
		for _, p := range readPolicies {
			if !p.permits(userValues, sessionValues) {
				return
			}
		}
		rates.Read += prob
		for _, p := range writePolicies {
			if !p.permits(userValues, sessionValues) {
				return
			}
		}
		rates.Write += prob
	})
	return rates
}

type attrDist struct {
	name      string
	isSession bool
	values    []AttributeValue
}

// forEachCombination calls fn for every combination of attribute values along
// with its probability. If there are too many combinations, it calls fn for a
// fixed number of random samples instead.
func (c Config) forEachCombination(fn func(prob float64, userValues, sessionValues map[string]string)) {
	// Only the attributes referenced by the policies affect the outcome.
	userRefs := make(map[string]bool)
	sessionRefs := make(map[string]bool)
	for _, p := range c.Policies {
		userRefs[p.UserAttribute] = true
		sessionRefs[p.SessionAttribute] = true
	}

	var dists []attrDist
	combinations := 1
	for _, attr := range c.UserAttributes {
		if userRefs[attr.Name] {
			dists = append(dists, attrDist{name: attr.Name, values: attr.Values})
			combinations = min(combinations*len(attr.Values), maxExactCombinations+1)
		}
	}
	for _, attr := range c.SessionAttributes {
		if sessionRefs[attr.Name] {
			dists = append(dists, attrDist{name: attr.Name, isSession: true, values: attr.Values})
			combinations = min(combinations*len(attr.Values), maxExactCombinations+1)
		}
	}

	userValues := make(map[string]string)
	sessionValues := make(map[string]string)
	set := func(d attrDist, value string) {
		if d.isSession {
			sessionValues[d.name] = value
		} else {
			userValues[d.name] = value
		}
	}

	if combinations > maxExactCombinations {
		rnd := rand.New(rand.NewSource(1))
		for range numRateSamples {
			for _, d := range dists {
				set(d, pickValueAt(rnd.Float64(), d.values))
			}
			fn(1.0/numRateSamples, userValues, sessionValues)
		}
		return
	}

	var walk func(i int, prob float64)
	walk = func(i int, prob float64) {
		if prob == 0 {
			return
		}
		if i == len(dists) {
			fn(prob, userValues, sessionValues)
			return
		}
		for _, v := range dists[i].values {
			set(dists[i], v.Value)
			walk(i+1, prob*v.Weight)
		}
	}
	walk(0, 1)
}
