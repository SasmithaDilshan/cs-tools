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

package flows

import (
	"encoding/json"
	"testing"

	"github.com/wso2-open-operations/cs-tools/integrations/csm-flow-service/internal/events"
)

// changedEvent builds an entity.changed event for a change request whose state
// moved from -> to, with an optional snapshot.
func changedEvent(t *testing.T, entityType, from, to string, snap map[string]any) Event {
	t.Helper()
	payload := events.EntityChangedPayload{
		EntityType: entityType,
		EntityID:   "cr-1",
		Changes:    map[string]map[string]any{},
		Snapshot:   snap,
	}
	if to != "" || from != "" {
		payload.Changes["state"] = map[string]any{"from": from, "to": to}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return Event{Envelope: events.Envelope{
		Type:     events.TypeEntityChanged,
		EntityID: "cr-1",
		Payload:  raw,
	}}
}

// TestCRApprovalNotice_Match pins the trigger. The ServiceNow condition is
// stateCHANGESTO-4^OR-3^OR5^OR0^OR1 -- CHANGES TO, not "is", which is the part
// a port gets wrong most easily: a change request updated for any other reason
// while already sitting in an approval state must not notify again.
func TestCRApprovalNotice_Match(t *testing.T) {
	f := crApprovalNotice{}

	cases := []struct {
		name string
		evt  Event
		want bool
	}{
		{"assess", changedEvent(t, "change_request", "NEW", "ASSESS", nil), true},
		{"authorize", changedEvent(t, "change_request", "ASSESS", "AUTHORIZE", nil), true},
		{"review", changedEvent(t, "change_request", "IMPLEMENT", "REVIEW", nil), true},
		{"customer approval", changedEvent(t, "change_request", "AUTHORIZE", "CUSTOMER_APPROVAL", nil), true},
		{"customer review", changedEvent(t, "change_request", "REVIEW", "CUSTOMER_REVIEW", nil), true},

		{"a state this flow does not cover", changedEvent(t, "change_request", "REVIEW", "CLOSED", nil), false},
		{"scheduled is not an approval state", changedEvent(t, "change_request", "ASSESS", "SCHEDULED", nil), false},
		{"no state change at all", changedEvent(t, "change_request", "", "", nil), false},
		{"already in the state — not a transition", changedEvent(t, "change_request", "ASSESS", "ASSESS", nil), false},
		{"a different entity", changedEvent(t, "case", "NEW", "ASSESS", nil), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := f.Match(tc.evt); got != tc.want {
				t.Errorf("Match = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCRApprovalNotice_MatchIgnoresOtherEventTypes proves the flow is inert for
// every event that is not entity.changed -- the registry runs every flow's
// Match against every record on the bus.
func TestCRApprovalNotice_MatchIgnoresOtherEventTypes(t *testing.T) {
	evt := changedEvent(t, "change_request", "NEW", "ASSESS", nil)
	for _, ty := range []events.Type{
		events.TypeCaseCreated, events.TypeCommentAdded, events.TypeIncidentCreated,
	} {
		evt.Envelope.Type = ty
		if (crApprovalNotice{}).Match(evt) {
			t.Errorf("matched %s, want no match", ty)
		}
	}
}

// TestCRApprovalNotice_MatchOnMalformedPayload proves a payload this flow
// cannot read is skipped rather than panicking the whole registry -- Match runs
// against every record, including ones written by services this one does not
// control.
func TestCRApprovalNotice_MatchOnMalformedPayload(t *testing.T) {
	evt := Event{Envelope: events.Envelope{
		Type:    events.TypeEntityChanged,
		Payload: json.RawMessage(`{"entityType": 12345}`),
	}}
	if (crApprovalNotice{}).Match(evt) {
		t.Error("expected no match on a payload that fails to decode")
	}
}

// TestCRSubject pins both subject shapes verbatim against the strings read out
// of ServiceNow's sys_element_mapping rows.
func TestCRSubject(t *testing.T) {
	cases := []struct{ number, suffix, team, want string }{
		{
			"CHG0031234", "Request for approval - Customer Review", "",
			"[WSO2 Support] [CR] (CHG0031234) Request for approval - Customer Review",
		},
		{
			"CHG0031234", "Request for Approval - Implementation", "",
			"[WSO2 Support] [CR] (CHG0031234) Request for Approval - Implementation",
		},
		{
			"CHG0031234", "Request for CAB approval - Authorize", "Choreo",
			"[WSO2 Support] [CR][Choreo] (CHG0031234) Request for CAB approval - Authorize",
		},
		{
			"CHG0031234", "Request for approval - Review", "MS",
			"[WSO2 Support] [CR][MS] (CHG0031234) Request for approval - Review",
		},
	}
	for _, tc := range cases {
		if got := crSubject(tc.number, tc.suffix, tc.team); got != tc.want {
			t.Errorf("crSubject(%q, %q, %q)\n got %q\nwant %q", tc.number, tc.suffix, tc.team, got, tc.want)
		}
	}
}

// TestCRTeamFromGitReference pins the If / Else If / Else chain, including its
// fallback: an unrecognised or absent reference is MS by design.
func TestCRTeamFromGitReference(t *testing.T) {
	cases := map[string]string{
		"https://github.com/wso2-enterprise/choreo-apis": "Choreo",
		"CHOREO-1234":                   "Choreo",
		"asgardeo/console":              "Asgardeo",
		"https://github.com/x/Asgardeo": "Asgardeo",
		"something-else":                "MS",
		"":                              "MS",
	}
	for ref, want := range cases {
		if got := crTeamFromGitReference(ref); got != want {
			t.Errorf("crTeamFromGitReference(%q) = %q, want %q", ref, got, want)
		}
	}
}

// TestCRApprovalStatesCoverTheTrigger is the guard that the branch table and
// the ServiceNow trigger condition stay in agreement. The original fires on
// exactly five states; a sixth added here without updating the trigger -- or
// one dropped -- is a silent behaviour change.
func TestCRApprovalStatesCoverTheTrigger(t *testing.T) {
	want := map[string]events.CRApprovalAudience{
		"ASSESS":            events.CRAudienceInternal,
		"AUTHORIZE":         events.CRAudienceInternal,
		"REVIEW":            events.CRAudienceInternal,
		"CUSTOMER_APPROVAL": events.CRAudienceCustomer,
		"CUSTOMER_REVIEW":   events.CRAudienceCustomer,
	}
	if len(crApprovalStates) != len(want) {
		t.Fatalf("branch table has %d states, the trigger covers %d", len(crApprovalStates), len(want))
	}
	for state, audience := range want {
		branch, ok := crApprovalStates[state]
		if !ok {
			t.Errorf("state %s missing from the branch table", state)
			continue
		}
		if branch.audience != audience {
			t.Errorf("state %s: audience %q, want %q", state, branch.audience, audience)
		}
		// Every internal branch names a group; no customer branch does.
		if audience == events.CRAudienceInternal && branch.group == "" {
			t.Errorf("state %s: internal branch with no approval group", state)
		}
		if audience == events.CRAudienceCustomer && branch.group != "" {
			t.Errorf("state %s: customer branch should name no group, got %q", state, branch.group)
		}
		if branch.suffix == "" {
			t.Errorf("state %s: empty subject suffix", state)
		}
	}
}

// TestNormaliseAddresses proves one person in two approval groups is notified
// once -- ServiceNow looped per recipient and could send duplicates.
func TestNormaliseAddresses(t *testing.T) {
	got := normaliseAddresses([]string{"B@wso2.com", "a@wso2.com", "", "  b@wso2.com  ", "a@wso2.com"})
	want := []string{"a@wso2.com", "b@wso2.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestCRApprovalNotice_NotRegistered guards the double-fire rule: registering
// this flow is a paired change with disabling its ServiceNow counterpart in the
// same commit (CLAUDE.md). ServiceNow still owns these notifications, so the
// flow must stay out of All() until that pairing happens -- and until
// csm-notification-service can consume what it publishes.
func TestCRApprovalNotice_NotRegistered(t *testing.T) {
	for _, f := range All() {
		if f.Key() == (crApprovalNotice{}).Key() {
			t.Fatal("cr_approval_notice is registered: registering it requires disabling the " +
				"ServiceNow flow in the same commit, and a csm-notification-service consumer " +
				"for change_request.approval_requested")
		}
	}
}

// TestBuildNoticeAppliesDebugRecipients proves EMAIL_DEBUG_RECIPIENTS replaces
// the real audience rather than adding to it, and — the part that matters — that
// it does NOT turn a would-be-silent event into mail.
func TestBuildNoticeAppliesDebugRecipients(t *testing.T) {
	t.Run("replaces the resolved audience entirely", func(t *testing.T) {
		real := []string{"devops-a@wso2.com", "devops-b@wso2.com"}
		debug := []string{"sasmitha@wso2.com"}

		got := applyDebugRecipients(real, debug)
		if len(got) != 1 || got[0] != "sasmitha@wso2.com" {
			t.Fatalf("got %v, want exactly [sasmitha@wso2.com] — the real audience must be replaced, not appended", got)
		}
	})

	t.Run("an empty override leaves the real audience alone", func(t *testing.T) {
		real := []string{"devops-a@wso2.com"}
		got := applyDebugRecipients(real, nil)
		if len(got) != 1 || got[0] != "devops-a@wso2.com" {
			t.Fatalf("got %v, want the real audience unchanged", got)
		}
	})

	t.Run("no real recipients stays silent even with an override set", func(t *testing.T) {
		got := applyDebugRecipients(nil, []string{"sasmitha@wso2.com"})
		if len(got) != 0 {
			t.Fatalf("got %v, want none — debug mode must not invent a notice nobody would have received", got)
		}
	})
}
