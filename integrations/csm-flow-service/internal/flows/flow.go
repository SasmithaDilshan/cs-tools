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

// EntityReader is the slice of entity-service a flow reads through. It starts
// at the two recipient lookups the first ported flow needs and grows with each
// port — never a speculative mirror of the whole entity API.
type EntityReader interface {
	// GroupMemberEmails returns the addresses of everyone in a named WSO2
	// group, e.g. "CAB Approval".
	GroupMemberEmails(ctx context.Context, groupName string) ([]string, error)
	// ProjectContactEmails returns a customer project's contact addresses.
	ProjectContactEmails(ctx context.Context, projectID string) ([]string, error)
}

// EventPublisher is the bus write a flow makes: one record, keyed so every
// event about the same entity stays ordered on one partition.
type EventPublisher interface {
	Publish(ctx context.Context, key, value []byte) error
}

// Deps are the shared clients a flow may use. A flow uses only what it needs;
// any of these may be nil in a deployment that hasn't configured it, so a flow
// that dereferences one is responsible for the entity being configured (or for
// letting the call fail cleanly).
type Deps struct {
	// Entity reads from entity-service — the recipient audiences a flow
	// resolves, and (for native entities) the records it writes.
	//
	// An interface, not *entity.Client, so a flow's Run is testable with a
	// fake: Run is where a port's real behaviour lives, and a concrete client
	// here would leave it exercisable only against a deployed service. Same
	// consumer-defined-narrow-interface shape the scheduled-tasks sub-crons
	// use (CaseSearcher, EmailSender). *entity.Client satisfies it; extend the
	// interface as each ported flow needs more, exactly as that client itself
	// grows.
	Entity EntityReader
	// Producer publishes back onto the bus: a notification-request event that
	// csm-notification-service sends, or (later) timer.fired from the sweeper.
	// An interface for the same reason as Entity; *eventbus.Producer satisfies it.
	Producer EventPublisher
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

// The production clients satisfy the interfaces above. These assertions are the
// guard that narrowing Deps for testability did not quietly fork the contract:
// add a method to EntityReader without adding it to *entity.Client and this
// stops compiling.
var (
	_ EntityReader   = (*entity.Client)(nil)
	_ EventPublisher = (*eventbus.Producer)(nil)
)
