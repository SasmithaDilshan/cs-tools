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

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wso2-open-operations/cs-tools/entity-service/internal/apierror"
	"github.com/wso2-open-operations/cs-tools/entity-service/internal/domain"
)

// --- fakes ---------------------------------------------------------------

type fakeQueryHourRepo struct {
	consumption  domain.ProjectConsumption
	consumptErr  error
	stored       *domain.ProjectQueryHours
	getErr       error
	upsertErr    error
	markPushErr  error
	staleIDs     []string
	staleErr     error
	timeCardProj string
	timeCardErr  error

	notifCtx domain.QueryHourNotificationContext
	notifErr error

	upsertedState int
	markedState   int
	markCalls     int
}

func (f *fakeQueryHourRepo) Consumption(_ context.Context, _ string) (domain.ProjectConsumption, error) {
	return f.consumption, f.consumptErr
}

func (f *fakeQueryHourRepo) Upsert(_ context.Context, c domain.ProjectConsumption, state int) (domain.ProjectQueryHours, error) {
	if f.upsertErr != nil {
		return domain.ProjectQueryHours{}, f.upsertErr
	}
	f.upsertedState = state
	out := domain.ProjectQueryHours{
		ProjectID:          c.ProjectID,
		ProjectKey:         c.ProjectKey,
		ProjectSFID:        c.ProjectSFID,
		EntitlementMinutes: c.EntitlementMinutes,
		ConsumedMinutes:    c.ConsumedMinutes(),
		BillableMinutes:    c.BillableMinutes,
		NonBillableMinutes: c.NonBillableMinutes,
		QueryHourState:     state,
		ComputedAt:         time.Now().UTC(),
	}
	if f.stored != nil {
		out.LastPushedState = f.stored.LastPushedState
		out.LastPushedAt = f.stored.LastPushedAt
	}
	return out, nil
}

func (f *fakeQueryHourRepo) MarkPushed(_ context.Context, _ string, state int, _ time.Time) error {
	f.markCalls++
	f.markedState = state
	return f.markPushErr
}

func (f *fakeQueryHourRepo) Get(_ context.Context, _ string) (domain.ProjectQueryHours, error) {
	if f.getErr != nil {
		return domain.ProjectQueryHours{}, f.getErr
	}
	if f.stored == nil {
		return domain.ProjectQueryHours{}, &apierror.NotFoundError{Msg: "none"}
	}
	return *f.stored, nil
}

func (f *fakeQueryHourRepo) StaleProjectIDs(_ context.Context, _ time.Time, _ int) ([]string, error) {
	return f.staleIDs, f.staleErr
}

func (f *fakeQueryHourRepo) ProjectIDForTimeCard(_ context.Context, _ string) (string, error) {
	return f.timeCardProj, f.timeCardErr
}

func (f *fakeQueryHourRepo) NotificationContext(_ context.Context, _ string) (domain.QueryHourNotificationContext, error) {
	return f.notifCtx, f.notifErr
}

type fakeNotifier struct {
	calls   int
	lastSub string
	lastPay domain.SubscriptionClosureUpdate
	err     error
}

func (f *fakeNotifier) NotifyClosureState(_ context.Context, sub string, p domain.SubscriptionClosureUpdate) error {
	f.calls++
	f.lastSub = sub
	f.lastPay = p
	return f.err
}

// queryHourStatePtr is local to these tests; the package already has an
// intPtr from sn_catalog_service_test.go.
func queryHourStatePtr(i int) *int { return &i }

// --- threshold mapping ----------------------------------------------------

// The thresholds are lifted from ServiceNow's `Set Project Query Hour State`.
// Boundaries are inclusive there (`>= 75`), so they are tested exactly.
func TestQueryHourStateFor_Thresholds(t *testing.T) {
	cases := []struct {
		name string
		pct  float64
		want int
	}{
		{"zero", 0, domain.QueryHourStateNormal},
		{"just under warning", 74.999, domain.QueryHourStateNormal},
		{"exactly warning", 75, domain.QueryHourStateWarning},
		{"between warning and critical", 80, domain.QueryHourStateWarning},
		{"exactly critical", 90, domain.QueryHourStateCritical},
		{"just under exceeded", 99.999, domain.QueryHourStateCritical},
		{"exactly exceeded", 100, domain.QueryHourStateExceeded},
		{"way over", 250, domain.QueryHourStateExceeded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := QueryHourStateFor(tc.pct); got != tc.want {
				t.Fatalf("QueryHourStateFor(%v) = %d, want %d", tc.pct, got, tc.want)
			}
		})
	}
}

// ServiceNow skipped projects with no entitlement to dodge a divide-by-zero.
// Here the project is still recorded, at 0%.
func TestRecompute_NoEntitlementIsZeroPercentNotADivideByZero(t *testing.T) {
	repo := &fakeQueryHourRepo{consumption: domain.ProjectConsumption{
		ProjectID: "p1", EntitlementMinutes: 0, BillableMinutes: 600,
	}}
	svc := NewQueryHourService(repo, nil, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if got.PercentConsumed != 0 {
		t.Fatalf("PercentConsumed = %v, want 0", got.PercentConsumed)
	}
	if got.QueryHourState != domain.QueryHourStateNormal {
		t.Fatalf("QueryHourState = %d, want %d", got.QueryHourState, domain.QueryHourStateNormal)
	}
}

// THE DIVERGENCE FROM SERVICENOW, PINNED.
// SN's rule is `if (newState > 0 && current != newState)`, so it never writes
// a lower state: once a project crossed 75% it stayed there even if the
// consumption was corrected downwards. This port recomputes honestly.
func TestRecompute_StateWalksBackDownWhenConsumptionDrops(t *testing.T) {
	repo := &fakeQueryHourRepo{
		// Previously at 3 (exceeded), and Choreo was told so.
		stored: &domain.ProjectQueryHours{
			QueryHourState: domain.QueryHourStateExceeded, LastPushedState: queryHourStatePtr(domain.QueryHourStateExceeded),
		},
		// A recalled time card has since dropped consumption to 50%.
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", ProjectSFID: "sf1",
			EntitlementMinutes: 6000, BillableMinutes: 3000,
		},
	}
	notifier := &fakeNotifier{}
	svc := NewQueryHourService(repo, notifier, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if got.QueryHourState != domain.QueryHourStateNormal {
		t.Fatalf("QueryHourState = %d, want %d (state must walk back down)",
			got.QueryHourState, domain.QueryHourStateNormal)
	}
	if !got.StateChanged {
		t.Fatal("StateChanged = false, want true")
	}
	if notifier.calls != 1 {
		t.Fatalf("notifier calls = %d, want 1 (a downward move must be pushed too)", notifier.calls)
	}
}

// Remaining minutes go negative on an overrun rather than clamping at zero —
// an overrun is real information and SN recorded it too.
func TestRecompute_OverrunReportsNegativeRemaining(t *testing.T) {
	repo := &fakeQueryHourRepo{consumption: domain.ProjectConsumption{
		ProjectID: "p1", EntitlementMinutes: 6000,
		BillableMinutes: 7000, NonBillableMinutes: 500,
	}}
	svc := NewQueryHourService(repo, nil, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if got.ConsumedMinutes != 7500 {
		t.Fatalf("ConsumedMinutes = %d, want 7500 (billable + non-billable)", got.ConsumedMinutes)
	}
	if got.RemainingMinutes != -1500 {
		t.Fatalf("RemainingMinutes = %d, want -1500", got.RemainingMinutes)
	}
	if got.QueryHourState != domain.QueryHourStateExceeded {
		t.Fatalf("QueryHourState = %d, want %d", got.QueryHourState, domain.QueryHourStateExceeded)
	}
}

// Re-running with unchanged numbers must not re-push: last_pushed_state
// already equals the computed state.
func TestRecompute_DoesNotRePushWhenStateUnchanged(t *testing.T) {
	repo := &fakeQueryHourRepo{
		stored: &domain.ProjectQueryHours{
			QueryHourState: domain.QueryHourStateWarning, LastPushedState: queryHourStatePtr(domain.QueryHourStateWarning),
		},
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", ProjectSFID: "sf1",
			EntitlementMinutes: 6000, BillableMinutes: 4800, // exactly 80%
		},
	}
	notifier := &fakeNotifier{}
	svc := NewQueryHourService(repo, notifier, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if notifier.calls != 0 {
		t.Fatalf("notifier calls = %d, want 0", notifier.calls)
	}
	if got.Pushed {
		t.Fatal("Pushed = true, want false")
	}
}

// A previously failed push is retried on the next recompute even when the
// state itself has not moved — that is what last_pushed_state is for.
func TestRecompute_RetriesPushAfterEarlierFailure(t *testing.T) {
	repo := &fakeQueryHourRepo{
		// State is 1, but Choreo was last told 0: the earlier push failed.
		stored: &domain.ProjectQueryHours{
			QueryHourState: domain.QueryHourStateWarning, LastPushedState: queryHourStatePtr(domain.QueryHourStateNormal),
		},
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", ProjectSFID: "sf1",
			EntitlementMinutes: 6000, BillableMinutes: 4800,
		},
	}
	notifier := &fakeNotifier{}
	svc := NewQueryHourService(repo, notifier, nil, true)

	if _, err := svc.Recompute(context.Background(), "p1"); err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if notifier.calls != 1 {
		t.Fatalf("notifier calls = %d, want 1 (failed push must retry)", notifier.calls)
	}
	if repo.markedState != domain.QueryHourStateWarning {
		t.Fatalf("markedState = %d, want %d", repo.markedState, domain.QueryHourStateWarning)
	}
}

// A push failure must not fail the recompute: the position is already stored
// and the next sweep retries. Same reasoning as the nil Event Hub publisher.
func TestRecompute_PushFailureDoesNotFailTheCall(t *testing.T) {
	repo := &fakeQueryHourRepo{consumption: domain.ProjectConsumption{
		ProjectID: "p1", ProjectSFID: "sf1",
		EntitlementMinutes: 6000, BillableMinutes: 6000,
	}}
	notifier := &fakeNotifier{err: errors.New("choreo 503")}
	svc := NewQueryHourService(repo, notifier, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute returned error %v, want nil", err)
	}
	if got.Pushed {
		t.Fatal("Pushed = true, want false")
	}
	if got.PushError == "" {
		t.Fatal("PushError is empty, want the failure reported")
	}
	if repo.markCalls != 0 {
		t.Fatalf("MarkPushed called %d times after a failed push, want 0", repo.markCalls)
	}
}

// A project with no Salesforce id cannot be pushed. SN would have sent
// `undefined` as the subscription id; skipping and saying so is the honest
// equivalent.
func TestRecompute_SkipsPushWhenProjectHasNoSalesforceID(t *testing.T) {
	repo := &fakeQueryHourRepo{consumption: domain.ProjectConsumption{
		ProjectID: "p1", ProjectSFID: "",
		EntitlementMinutes: 6000, BillableMinutes: 6000,
	}}
	notifier := &fakeNotifier{}
	svc := NewQueryHourService(repo, notifier, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if notifier.calls != 0 {
		t.Fatalf("notifier calls = %d, want 0", notifier.calls)
	}
	if got.PushError == "" {
		t.Fatal("PushError is empty, want the skip reported")
	}
}

// A nil notifier (QUERY_HOUR_CHOREO_BASE_URL unset) disables pushing without
// disabling the recompute, and must not panic.
func TestRecompute_NilNotifierStillRecords(t *testing.T) {
	repo := &fakeQueryHourRepo{consumption: domain.ProjectConsumption{
		ProjectID: "p1", ProjectSFID: "sf1",
		EntitlementMinutes: 6000, BillableMinutes: 6000,
	}}
	svc := NewQueryHourService(repo, nil, nil, true)

	got, err := svc.Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if got.QueryHourState != domain.QueryHourStateExceeded {
		t.Fatalf("QueryHourState = %d, want %d", got.QueryHourState, domain.QueryHourStateExceeded)
	}
	if got.Pushed {
		t.Fatal("Pushed = true, want false")
	}
}

// The payload field names and units are what the Choreo service already
// receives from ServiceNow, so cutover needs no change on their side.
func TestRecompute_PushPayloadMatchesServiceNowShape(t *testing.T) {
	repo := &fakeQueryHourRepo{consumption: domain.ProjectConsumption{
		ProjectID: "p1", ProjectSFID: "a0d7h00000Dt0R4AAJ",
		EntitlementMinutes: 12600, BillableMinutes: 19281,
	}}
	notifier := &fakeNotifier{}
	svc := NewQueryHourService(repo, notifier, nil, true)

	if _, err := svc.Recompute(context.Background(), "p1"); err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if notifier.lastSub != "a0d7h00000Dt0R4AAJ" {
		t.Fatalf("subscriptionId = %q, want the project's Salesforce id", notifier.lastSub)
	}
	if notifier.lastPay.ConsumedQueryTime != 19281 || notifier.lastPay.TotalQueryTime != 12600 {
		t.Fatalf("payload = %+v, want consumed 19281 / total 12600 (minutes)", notifier.lastPay)
	}
}

// THE OTHER DIVERGENCE, PINNED: the flow's account-wide fan-out is gone.
// One time card resolves to exactly one project.
func TestRecomputeForTimeCard_ScopesToTheCardsOwnProject(t *testing.T) {
	repo := &fakeQueryHourRepo{
		timeCardProj: "p-owning",
		consumption: domain.ProjectConsumption{
			ProjectID: "p-owning", EntitlementMinutes: 6000, BillableMinutes: 600,
		},
	}
	svc := NewQueryHourService(repo, nil, nil, true)

	got, err := svc.RecomputeForTimeCard(context.Background(), "tc1")
	if err != nil {
		t.Fatalf("RecomputeForTimeCard: %v", err)
	}
	if got.ProjectID != "p-owning" {
		t.Fatalf("ProjectID = %q, want p-owning", got.ProjectID)
	}
}

func TestRecomputeForTimeCard_PropagatesNotFound(t *testing.T) {
	repo := &fakeQueryHourRepo{timeCardErr: &apierror.NotFoundError{Msg: "time card missing"}}
	svc := NewQueryHourService(repo, nil, nil, true)

	_, err := svc.RecomputeForTimeCard(context.Background(), "nope")
	var nfe *apierror.NotFoundError
	if !errors.As(err, &nfe) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
}

func TestRecompute_RejectsEmptyProjectID(t *testing.T) {
	svc := NewQueryHourService(&fakeQueryHourRepo{}, nil, nil, true)
	_, err := svc.Recompute(context.Background(), "   ")
	var ve *apierror.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}

// One bad project must not abandon the sweep: it stays the stalest, so the
// next run would hit it first and stall there forever.
func TestSweep_ContinuesPastAFailingProject(t *testing.T) {
	repo := &failOnFirstRepo{
		fakeQueryHourRepo: fakeQueryHourRepo{
			staleIDs: []string{"bad", "good1", "good2"},
			consumption: domain.ProjectConsumption{
				EntitlementMinutes: 6000, BillableMinutes: 600,
			},
		},
		failFor: "bad",
	}
	svc := NewQueryHourService(repo, nil, nil, true)

	got, err := svc.Sweep(context.Background(), time.Hour, 10)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if got.Requested != 3 || got.Succeeded != 2 || got.Failed != 1 {
		t.Fatalf("got requested=%d succeeded=%d failed=%d, want 3/2/1",
			got.Requested, got.Succeeded, got.Failed)
	}
	if _, ok := got.Errors["bad"]; !ok {
		t.Fatalf("Errors = %v, want an entry for the failing project", got.Errors)
	}
}

func TestSweep_RejectsLimitAboveMaximum(t *testing.T) {
	svc := NewQueryHourService(&fakeQueryHourRepo{}, nil, nil, true)
	_, err := svc.Sweep(context.Background(), time.Hour, maxSweepLimit+1)
	var ve *apierror.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}

func TestSweep_RejectsNegativeStaleFor(t *testing.T) {
	svc := NewQueryHourService(&fakeQueryHourRepo{}, nil, nil, true)
	_, err := svc.Sweep(context.Background(), -time.Minute, 10)
	var ve *apierror.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}

// failOnFirstRepo makes Consumption fail for one nominated project id.
type failOnFirstRepo struct {
	fakeQueryHourRepo
	failFor string
}

func (f *failOnFirstRepo) Consumption(ctx context.Context, projectID string) (domain.ProjectConsumption, error) {
	if projectID == f.failFor {
		return domain.ProjectConsumption{}, errors.New("boom")
	}
	c := f.fakeQueryHourRepo.consumption
	c.ProjectID = projectID
	return c, nil
}

// --- review findings, pinned -------------------------------------------

// Any error from Get other than NotFound must propagate. Treating a timeout
// as "no previous state" would re-notify a project already at its state.
func TestRecompute_PropagatesNonNotFoundErrorFromGet(t *testing.T) {
	repo := &fakeQueryHourRepo{
		getErr: errors.New("connection reset"),
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", EntitlementMinutes: 6000, BillableMinutes: 6000,
		},
	}
	_, err := NewQueryHourService(repo, nil, nil, true).Recompute(context.Background(), "p1")
	if err == nil {
		t.Fatal("Recompute returned nil, want the transport error propagated")
	}
}

// A project with no stored row is a FIRST computation, not a crossing. Without
// this, the first sweep after deploy emails every project already past 75% —
// notices ServiceNow has already sent.
func TestRecompute_FirstComputationIsABaselineAndDoesNotNotify(t *testing.T) {
	repo := &fakeQueryHourRepo{
		stored: nil, // no row yet
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", ProjectSFID: "sf1",
			EntitlementMinutes: 6000, BillableMinutes: 6000, // 100% -> state 3
		},
	}
	pub := &fakePublisher{}
	got, err := NewQueryHourService(repo, nil, queryHourPublisher{pub}, true).Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if got.QueryHourState != domain.QueryHourStateExceeded {
		t.Fatalf("state = %d, want %d — the position must still be recorded",
			got.QueryHourState, domain.QueryHourStateExceeded)
	}
	if len(pub.sent) != 0 {
		t.Fatalf("published %d events on a first computation, want 0", len(pub.sent))
	}
}

// The second recompute onwards can notify, once a baseline exists.
func TestRecompute_NotifiesOnceABaselineExists(t *testing.T) {
	repo := &fakeQueryHourRepo{
		stored: &domain.ProjectQueryHours{QueryHourState: domain.QueryHourStateNormal},
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", ProjectSFID: "sf1",
			EntitlementMinutes: 6000, BillableMinutes: 4500, // 75% -> state 1
		},
	}
	pub := &fakePublisher{}
	if _, err := NewQueryHourService(repo, nil, queryHourPublisher{pub}, true).Recompute(context.Background(), "p1"); err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(pub.sent) != 1 {
		t.Fatalf("published %d events, want 1", len(pub.sent))
	}
}

// The notification gate is independent of Event Hub: with it off, a real
// crossing records and pushes but never emails. This is what makes the
// parallel run alongside ServiceNow actually silent.
func TestRecompute_NotificationsDisabledSuppressesTheEmailOnly(t *testing.T) {
	repo := &fakeQueryHourRepo{
		stored: &domain.ProjectQueryHours{QueryHourState: domain.QueryHourStateNormal},
		consumption: domain.ProjectConsumption{
			ProjectID: "p1", ProjectSFID: "sf1",
			EntitlementMinutes: 6000, BillableMinutes: 4500,
		},
	}
	pub := &fakePublisher{}
	notifier := &fakeNotifier{}
	got, err := NewQueryHourService(repo, notifier, queryHourPublisher{pub}, false).Recompute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(pub.sent) != 0 {
		t.Fatalf("published %d events with notifications disabled, want 0", len(pub.sent))
	}
	if notifier.calls != 1 {
		t.Fatalf("Choreo push calls = %d, want 1 — the gate must not affect the push", notifier.calls)
	}
	if got.QueryHourState != domain.QueryHourStateWarning {
		t.Fatalf("state = %d, want %d — the position must still be recorded",
			got.QueryHourState, domain.QueryHourStateWarning)
	}
}

// A cancelled request context stops the sweep cleanly rather than converting
// every remaining project into a spurious failure.
func TestSweep_StopsCleanlyWhenTheDeadlineIsReached(t *testing.T) {
	repo := &fakeQueryHourRepo{
		staleIDs:    []string{"a", "b", "c"},
		consumption: domain.ProjectConsumption{EntitlementMinutes: 6000, BillableMinutes: 600},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already past the deadline

	got, err := NewQueryHourService(repo, nil, nil, true).Sweep(ctx, time.Hour, 10)
	if err != nil {
		t.Fatalf("Sweep returned %v, want a clean partial result", err)
	}
	if got.Failed != 0 {
		t.Fatalf("Failed = %d, want 0 — a deadline is not a per-project failure", got.Failed)
	}
	if got.Requested != 0 {
		t.Fatalf("Requested = %d, want 0 attempted", got.Requested)
	}
}

// queryHourPublisher adapts the package's fakePublisher to
// EventPublisherService, which also requires Close. Defined here rather than
// adding Close to fakePublisher so cr_notice_service_test.go is untouched.
type queryHourPublisher struct{ *fakePublisher }

func (queryHourPublisher) Close() {}

// The sweep must finish inside the server's 15s WriteTimeout, not the 30s
// request context — otherwise it does the work and cannot deliver it. This
// pins the budget as the binding constraint.
func TestSweep_BudgetIsInsideTheServerWriteDeadline(t *testing.T) {
	const serverWriteTimeout = 15 * time.Second // entity-service/internal/server/server.go
	if sweepBudget >= serverWriteTimeout {
		t.Fatalf("sweepBudget %v must leave headroom inside the %v write deadline",
			sweepBudget, serverWriteTimeout)
	}
	// Enough room left to serialise and write a full batch response.
	if margin := serverWriteTimeout - sweepBudget; margin < 3*time.Second {
		t.Fatalf("only %v left to write the response; want at least 3s", margin)
	}
}

// A slow project must not let the sweep run past its budget.
func TestSweep_StopsOnTheTimeBudget(t *testing.T) {
	// Shrink the budget so the test is fast; restore it afterwards.
	original := sweepBudget
	sweepBudget = 40 * time.Millisecond
	defer func() { sweepBudget = original }()

	repo := &slowRepo{
		fakeQueryHourRepo: fakeQueryHourRepo{
			staleIDs:    []string{"a", "b", "c", "d"},
			consumption: domain.ProjectConsumption{EntitlementMinutes: 6000, BillableMinutes: 600},
		},
		delay: 25 * time.Millisecond,
	}
	got, err := NewQueryHourService(repo, nil, nil, true).Sweep(context.Background(), time.Hour, 10)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	// Two projects consume the budget; the rest must be left for the next run.
	if got.Requested >= 4 {
		t.Fatalf("Requested = %d, want fewer than 4 — the budget should have cut it short", got.Requested)
	}
	if got.Failed != 0 {
		t.Fatalf("Failed = %d, want 0 — a budget stop is not a per-project failure", got.Failed)
	}
}

// slowRepo makes each Consumption call take a fixed time.
type slowRepo struct {
	fakeQueryHourRepo
	delay time.Duration
}

func (r *slowRepo) Consumption(_ context.Context, projectID string) (domain.ProjectConsumption, error) {
	time.Sleep(r.delay)
	c := r.fakeQueryHourRepo.consumption
	c.ProjectID = projectID
	return c, nil
}
