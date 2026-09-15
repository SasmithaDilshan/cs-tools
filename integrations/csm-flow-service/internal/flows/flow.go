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

// Package flows holds the hand-ported ServiceNow flows, one Go type per flow
// (or per collapsed family), plus the Registry that routes each consumed event
// to the flows that match it.
//
// This is the pragmatic-hand-port shape (docs/cutover-porting-plan.md §2): the
// configurable engine (spec tree, compiler, spec DB tables) is deferred, so a
// flow is Go code implementing Flow, not a row of JSON. The condition
// evaluator (internal/eval) and the ServiceNow-query translator
// (internal/porting/snquery) remain available — a flow may express its trigger
// condition as a spec.Condition and evaluate it with eval, or just check
// fields directly in Match; both are fine.
//
// Every ported flow's real trigger, condition and actions are recorded in
// docs/flow-porting-specs.md — that is the source of truth a port is written
// from and verified against.
package flows

import (
	"context"

	"github.com/wso2-open-operations/cs-tools/integrations/csm-flow-service/internal/entity"
	"github.com/wso2-open-operations/cs-tools/integrations/csm-flow-service/internal/eventbus"
	"github.com/wso2-open-operations/cs-tools/integrations/csm-flow-service/internal/events"
)

// Deps are the shared clients a flow may use. A flow uses only what it needs;
// any of these may be nil in a deployment that hasn't configured it, so a flow
// that dereferences one is responsible for the entity being configured (or for
// letting the call fail cleanly).
type Deps struct {
	// Entity is the entity-service client — reads and (for native entities)
	// writes cases and related records.
	Entity *entity.Client
	// Producer publishes back onto the bus: a notification-request event that
	// csm-notification-service sends, or (later) timer.fired from the sweeper.
	Producer *eventbus.Producer
	// EmailDebugRecipients, when non-empty, replaces the real audience of every
	// notification a flow requests — approval groups, project contacts,
	// watchers — so a dev or staging deployment can be exercised without mail
	// reaching real people. See config.Config.EmailDebugRecipients.
	//
	// A flow honouring this must still RESOLVE its real recipients first and
	// swap only the final list: that keeps a broken entity-service lookup
	// visible instead of masked, and means a flow with no real audience still
	// sends nothing rather than mailing the debug list about an event nobody
	// would have been told about.
	EmailDebugRecipients []string
}

// Event is a decoded bus record handed to a flow.
type Event struct {
	// Envelope is the parsed wire envelope: Type, EntityID, raw Payload.
	Envelope events.Envelope
	// Record is the underlying bus record, for idempotency coordinates and
	// the raw bytes.
	Record eventbus.Record
}

// Flow is one ported ServiceNow flow.
//
//   - Key is a stable identifier (e.g. "watch_list"). It names the flow in
//     logs and is half of the (event, flow) idempotency key once the dispatch
//     log lands (docs/architecture.md §9). Renaming one is a semantic change.
//   - Match is the trigger + condition gate. It MUST be pure (no I/O): it
//     decides, from the event alone, whether this flow reacts. Keeping it pure
//     is what makes the routing cheap and exhaustively testable.
//   - Run performs the flow's actions. A non-nil error causes the record to be
//     retried (eventbus.handleAttempts), so Run must be safe to re-run — see
//     the idempotency note on Registry.Handle.
type Flow interface {
	Key() string
	Match(evt Event) bool
	Run(ctx context.Context, evt Event, deps Deps) error
}
