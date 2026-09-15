// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package config loads the flow engine's configuration from the environment.
// The event bus is mandatory (mustEnv — the consumer's whole reason to exist),
// while entity-service and the DLQ are optional at construction and only fail
// on first use, matching the repo's config-strictness convention
// (docs/architecture.md §17.3). Every value is a flat single-line string:
// Choreo's config UI cannot deploy nested collections, which is also why flow
// definitions live in Postgres, not config.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Defaults.
const (
	defaultPort          = "8080"
	defaultConsumerGroup = "csm-flow-service"
)

// Config is the fully-resolved configuration.
type Config struct {
	// Event bus (required).
	EventHubBroker           string
	EventHubConnectionString string
	EventHubTopic            string
	ConsumerGroup            string

	// Dead-letter topic (optional). When empty, a record that exhausts its
	// retries is logged and dropped rather than dead-lettered.
	DLQTopic string

	// entity-service client (optional at startup; required by any flow that
	// calls it). Credentials are the shared OAuth2 app.
	EntityBaseURL string
	EntityScopes  []string
	OAuthClientID string
	OAuthSecret   string
	OAuthTokenURL string

	// HTTP health/metrics server.
	Port string
}

// Load reads and validates configuration from the environment.
func Load() (Config, error) {
	var missing []string
	must := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := Config{
		EventHubBroker:           must("EVENT_HUB_BROKER"),
		EventHubConnectionString: must("EVENT_HUB_CONNECTION_STRING"),
		EventHubTopic:            must("EVENT_HUB_TOPIC"),
		ConsumerGroup:            getenv("EVENT_HUB_CONSUMER_GROUP", defaultConsumerGroup),
		DLQTopic:                 strings.TrimSpace(os.Getenv("EVENT_HUB_DLQ_TOPIC")),

		EntityBaseURL: strings.TrimSpace(os.Getenv("CUSTOMER_ENTITY_BASE_URL")),
		EntityScopes:  splitScopes(os.Getenv("CUSTOMER_ENTITY_SCOPES")),
		OAuthClientID: strings.TrimSpace(os.Getenv("OAUTH2_CLIENT_ID")),
		OAuthSecret:   strings.TrimSpace(os.Getenv("OAUTH2_CLIENT_SECRET")),
		OAuthTokenURL: strings.TrimSpace(os.Getenv("OAUTH2_TOKEN_URL")),

		Port: getenv("PORT", defaultPort),
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: missing required env vars: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// HasEntity reports whether the entity-service client is configured. A flow
// that needs entity-service should check this (or simply let the call fail) —
// the client is safe to construct either way.
func (c Config) HasEntity() bool {
	return c.EntityBaseURL != "" && c.OAuthClientID != "" && c.OAuthTokenURL != ""
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// splitScopes parses a space- or comma-separated scope list into a slice.
func splitScopes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ' ' || r == ',' })
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// atoiOr is a small helper for optional numeric env vars (kept for the
// consumer-count knobs added with Phase 1 scaling).
func atoiOr(raw string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
		return n
	}
	return def
}

// ensure atoiOr is retained even before its first caller lands.
var _ = atoiOr
