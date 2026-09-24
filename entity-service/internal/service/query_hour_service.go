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
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/wso2-open-operations/cs-tools/entity-service/internal/apierror"
	"github.com/wso2-open-operations/cs-tools/entity-service/internal/domain"
	"github.com/wso2-open-operations/cs-tools/entity-service/internal/events"
	"github.com/wso2-open-operations/cs-tools/entity-service/internal/repository"
)

// firstRecompute is the previousState sentinel for a project with no stored
// position yet. Distinct from state 0 ("computed, below every threshold"),
// which is a real state that a later crossing can rise from.
const firstRecompute = -1

// defaultSweepLimit caps one scheduled sweep. The sweep is resumable — it
// orders by staleness — so a cap costs latency, never coverage.
const defaultSweepLimit = 200

// maxSweepLimit bounds what a caller may ask for in one request, so a typo in
// the scheduled task's config cannot turn into an unbounded scan.
const maxSweepLimit = 2000

// SubscriptionClosureNotifier pushes a project's consumption to Choreo Sales
// Operations. Ported from the `Consumed Query Hour Update` business rule,
// which calls REST message "Choreo API Sales Operations" /
// "Update Subscription Closure State" keyed by the project's Salesforce id.
//
// A nil notifier means pushing is disabled; the recompute still runs and is
// still recorded. Losing the outbound call is strictly better than refusing
// to record the position, and last_pushed_state makes the next recompute
// retry it.
type SubscriptionClosureNotifier interface {
	// NotifyClosureState pushes one project's position. subscriptionID is the
	// project's Salesforce id (project.sf_id / SN's u_project_id).
	NotifyClosureState(ctx context.Context, subscriptionID string, payload domain.SubscriptionClosureUpdate) error
}

// queryHourCcGroups are the standing internal recipients of every threshold
// notice, hardcoded in ServiceNow's flow script and kept hardcoded here for
// the same reason: they are a policy about who watches query-hour burn, not a
// per-deployment setting, and moving them to config would make it possible to
// ship a build that quietly tells nobody.
var queryHourCcGroups = []string{
	"cs-management-group@wso2.com",
	"bizdev@wso2.com",
	"cs-tooling-notification-group@wso2.com",
}

// queryHourExceededCc is added only at state 3. ServiceNow did exactly this
// (`cc_list.push("ruwan@wso2.com")` inside the state==3 branch).
const queryHourExceededCc = "ruwan@wso2.com"

// internalEmailDomain gates the recipient list. ServiceNow checked
// `ref_email.includes("@wso2.com")` on each resolved address before adding it.
const internalEmailDomain = "@wso2.com"

type queryHourService struct {
	repo     repository.QueryHourRepository
	notifier SubscriptionClosureNotifier
	// publisher is nil when Event Hub is not configured, in which case the
	// position is still recomputed and stored and only the email is skipped —
	// the same convention as engagementAllocationService.publisher.
	publisher EventPublisherService
	// notificationsEnabled gates the threshold email separately from Event
	// Hub. Off by default: see config.QueryHourNotificationsEnabled for why
	// the Choreo kill switch alone was not enough.
	notificationsEnabled bool
}

// NewQueryHourService constructs a QueryHourService. notifier may be nil —
// see SubscriptionClosureNotifier.
func NewQueryHourService(
	repo repository.QueryHourRepository,
	notifier SubscriptionClosureNotifier,
	publisher EventPublisherService,
	notificationsEnabled bool,
) QueryHourService {
	return &queryHourService{
		repo:                 repo,
		notifier:             notifier,
		publisher:            publisher,
		notificationsEnabled: notificationsEnabled,
	}
}

// QueryHourStateFor maps percent-consumed to ServiceNow's u_query_hour_state.
//
// Unlike ServiceNow's `Set Project Query Hour State`, which guards its write
// with `if (newState > 0 && ...)` and therefore only ever RAISES the state,
// this returns the honest state for the current numbers. A recalled or
// corrected time card lowers it again. That is a deliberate divergence.
func QueryHourStateFor(percentConsumed float64) int {
	for _, t := range domain.QueryHourThresholds {
		if percentConsumed >= t.MinPercent {
			return t.State
		}
	}
	return domain.QueryHourStateNormal
}

// percentConsumed is consumed as a percentage of entitlement, or 0 when there
// is no entitlement to divide by. ServiceNow skipped such projects outright
// (`if (totalHours <= 0) continue`) to avoid a divide-by-zero; recording a
// zero-percent row instead means the project still appears in the data with
// an explicit "no entitlement" reading rather than silently missing.
func percentConsumed(consumed, entitlement int) float64 {
	if entitlement <= 0 {
		return 0
	}
	return (float64(consumed) / float64(entitlement)) * 100
}

func (s *queryHourService) Recompute(ctx context.Context, projectID string) (domain.RecomputeQueryHoursResponse, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.RecomputeQueryHoursResponse{}, &apierror.ValidationError{Msg: "projectId is required"}
	}

	// The previous state is read before the recompute so StateChanged can be
	// reported, and so a first computation can be told apart from a real
	// crossing.
	//
	// ONLY a NotFoundError may be swallowed here. Treating every error as
	// "no previous state" would make a timeout or a dropped connection look
	// like a first recompute, which re-notifies a project already sitting at
	// its current state and reports StateChanged=false for a real move.
	previousState := firstRecompute
	if prev, err := s.repo.Get(ctx, projectID); err == nil {
		previousState = prev.QueryHourState
	} else {
		var nfe *apierror.NotFoundError
		if !errors.As(err, &nfe) {
			return domain.RecomputeQueryHoursResponse{}, err
		}
	}

	consumption, err := s.repo.Consumption(ctx, projectID)
	if err != nil {
		return domain.RecomputeQueryHoursResponse{}, err
	}

	consumed := consumption.ConsumedMinutes()
	pct := percentConsumed(consumed, consumption.EntitlementMinutes)
	state := QueryHourStateFor(pct)

	stored, err := s.repo.Upsert(ctx, consumption, state)
	if err != nil {
		return domain.RecomputeQueryHoursResponse{}, err
	}
	stored.PercentConsumed = pct
	stored.RemainingMinutes = consumption.EntitlementMinutes - consumed
	stored.EntitlementSource = consumption.EntitlementSource
	stored.SyncedEntitlementMinutes = consumption.SyncedEntitlementMinutes
	stored.UnmatchedLineCount = consumption.UnmatchedLineCount

	// A product line the entitlement rule does not recognise contributes zero
	// hours and says nothing about it — ServiceNow's behaviour, faithfully
	// kept. Log it so the silence is at least visible in one place.
	if consumption.UnmatchedLineCount > 0 {
		slog.Warn("query hours: active opportunity lines with an unrecognised product name contributed zero entitlement",
			"projectId", projectID, "unmatchedLines", consumption.UnmatchedLineCount,
			"activeLines", consumption.ActiveLineCount)
	}
	// During the parallel run, a derived figure that disagrees with
	// ServiceNow's is the thing worth seeing before cutover.
	if consumption.EntitlementSource == domain.EntitlementSourceOpportunityLines &&
		consumption.SyncedEntitlementMinutes != consumption.EntitlementMinutes {
		slog.Info("query hours: derived entitlement differs from the ServiceNow figure",
			"projectId", projectID,
			"derivedMinutes", consumption.EntitlementMinutes,
			"serviceNowMinutes", consumption.SyncedEntitlementMinutes)
	}

	resp := domain.RecomputeQueryHoursResponse{
		ProjectQueryHours: stored,
		StateChanged:      previousState >= 0 && previousState != state,
	}

	// Notify only on an UPWARD crossing into a real threshold. Going down is
	// not news anyone needs mailing about, and state 0 has no message at all —
	// ServiceNow built an email for it anyway, with the literal word
	// "undefined" in the body, because its `internal_message` variable was
	// never assigned on that path.
	//
	// A FIRST computation is never a crossing. Without this guard the very
	// first sweep after deploy would email the owners and the cc groups for
	// every project already past 75% — notices ServiceNow has already sent.
	// The first recompute records a baseline silently; the second one onwards
	// can notify.
	if previousState != firstRecompute && state > previousState && state >= domain.QueryHourStateWarning {
		s.publishThresholdReached(ctx, projectID, state, previousState, consumption, stored)
	} else if previousState == firstRecompute && state >= domain.QueryHourStateWarning {
		slog.InfoContext(ctx, "query hours: first computation recorded as a baseline, not notified",
			"projectId", projectID, "state", state)
	}

	// Push only when the state Choreo last accepted differs from the current
	// one. That covers both "it moved" and "a previous push failed", and
	// makes a repeated recompute with unchanged numbers a no-op.
	if stored.LastPushedState != nil && *stored.LastPushedState == state {
		return resp, nil
	}
	if s.notifier == nil {
		slog.Debug("query-hour push skipped: notifier not configured",
			"projectId", projectID, "state", state)
		return resp, nil
	}
	if consumption.ProjectSFID == "" {
		// SN keyed the call on u_project_id and would have sent `undefined`
		// for a project without one. Skipping is the honest equivalent.
		slog.Warn("query-hour push skipped: project has no Salesforce id",
			"projectId", projectID, "state", state)
		resp.PushError = "project has no Salesforce id"
		return resp, nil
	}

	pushErr := s.notifier.NotifyClosureState(ctx, consumption.ProjectSFID, domain.SubscriptionClosureUpdate{
		ConsumedQueryTime: consumed,
		TotalQueryTime:    consumption.EntitlementMinutes,
	})
	if pushErr != nil {
		// A failed push must not fail the recompute: the position is already
		// stored, and last_pushed_state still differs, so the next sweep
		// retries. Same reasoning as the nil publisher in
		// engagementAllocationService.
		slog.Error("query-hour push to Choreo failed",
			"projectId", projectID, "sfId", consumption.ProjectSFID,
			"state", state, "error", pushErr)
		resp.PushError = pushErr.Error()
		return resp, nil
	}

	now := time.Now().UTC()
	if err := s.repo.MarkPushed(ctx, projectID, state, now); err != nil {
		// The push landed but we failed to record it. Report success — the
		// worst case is one duplicate push next sweep, which the receiving
		// Choreo service treats as idempotent (it sets a state, not a delta).
		slog.Error("query-hour push succeeded but could not be recorded",
			"projectId", projectID, "state", state, "error", err)
	} else {
		resp.LastPushedState = &state
		resp.LastPushedAt = &now
	}
	resp.Pushed = true
	return resp, nil
}

func (s *queryHourService) RecomputeForTimeCard(ctx context.Context, timeCardID string) (domain.RecomputeQueryHoursResponse, error) {
	timeCardID = strings.TrimSpace(timeCardID)
	if timeCardID == "" {
		return domain.RecomputeQueryHoursResponse{}, &apierror.ValidationError{Msg: "timeCardId is required"}
	}
	projectID, err := s.repo.ProjectIDForTimeCard(ctx, timeCardID)
	if err != nil {
		return domain.RecomputeQueryHoursResponse{}, err
	}
	// Scoped to the time card's OWN project. ServiceNow's flow instead looked
	// every project under the case's ACCOUNT up (max 1000) and recomputed all
	// of them on every approval; that fan-out is not reproduced.
	return s.Recompute(ctx, projectID)
}

func (s *queryHourService) Get(ctx context.Context, projectID string) (domain.ProjectQueryHours, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ProjectQueryHours{}, &apierror.ValidationError{Msg: "projectId is required"}
	}
	q, err := s.repo.Get(ctx, projectID)
	if err != nil {
		return domain.ProjectQueryHours{}, err
	}
	q.PercentConsumed = percentConsumed(q.ConsumedMinutes, q.EntitlementMinutes)
	q.RemainingMinutes = q.EntitlementMinutes - q.ConsumedMinutes
	return q, nil
}

func (s *queryHourService) Sweep(ctx context.Context, staleFor time.Duration, limit int) (domain.RecomputeQueryHoursBatchResponse, error) {
	if limit <= 0 {
		limit = defaultSweepLimit
	}
	if limit > maxSweepLimit {
		return domain.RecomputeQueryHoursBatchResponse{}, &apierror.ValidationError{
			Msg: "limit exceeds the maximum of 2000"}
	}
	if staleFor < 0 {
		return domain.RecomputeQueryHoursBatchResponse{}, &apierror.ValidationError{
			Msg: "staleFor must not be negative"}
	}

	cutoff := time.Now().UTC().Add(-staleFor)
	ids, err := s.repo.StaleProjectIDs(ctx, cutoff, limit)
	if err != nil {
		return domain.RecomputeQueryHoursBatchResponse{}, err
	}

	out := domain.RecomputeQueryHoursBatchResponse{Requested: len(ids)}
	for _, id := range ids {
		// The sweep runs inside an HTTP request, and routes.go wraps the mux
		// in middleware.Timeout(30s). A long sweep would otherwise keep
		// calling Recompute with an already-cancelled context, turning every
		// remaining project into a spurious failure and burning the whole
		// budget on errors.
		//
		// Stopping cleanly instead leaves the unprocessed projects untouched,
		// so they stay the stalest and the next run picks them up first.
		// Requested is corrected to what was actually attempted, so a caller
		// can see the sweep was cut short rather than inferring it.
		if ctx.Err() != nil {
			attempted := out.Succeeded + out.Failed
			slog.WarnContext(ctx, "query-hour sweep stopped early: request deadline reached",
				"succeeded", out.Succeeded, "failed", out.Failed,
				"notAttempted", len(ids)-attempted)
			out.Requested = attempted
			return out, nil
		}
		// One project's failure must not abandon the rest of the sweep: the
		// next run would hit the same project first (it stays stalest) and
		// stall forever. Record and continue.
		res, err := s.Recompute(ctx, id)
		if err != nil {
			out.Failed++
			if out.Errors == nil {
				out.Errors = make(map[string]string)
			}
			out.Errors[id] = err.Error()
			slog.Error("query-hour sweep: project failed", "projectId", id, "error", err)
			continue
		}
		out.Succeeded++
		out.Results = append(out.Results, res)
	}
	return out, nil
}

// publishThresholdReached emails the account manager and technical owner that
// a project has crossed 75%, 90% or 100%.
//
// Port of ServiceNow's `[WSO2][Query Hour] Usage Notifications - Project`. All
// recipient resolution and subject rendering happen here, before the event is
// published, so csm-notification-service formats and sends and decides
// nothing — the division every other notification in this service uses.
//
// A failure anywhere in here is logged and swallowed: the recompute has
// already stored the position and already pushed to Choreo, and losing one
// email is strictly better than failing a run that otherwise succeeded.
func (s *queryHourService) publishThresholdReached(
	ctx context.Context,
	projectID string,
	state, previousState int,
	consumption domain.ProjectConsumption,
	stored domain.ProjectQueryHours,
) {
	if !s.notificationsEnabled {
		slog.InfoContext(ctx, "query-hour threshold reached but notifications are disabled; nothing emailed",
			"projectId", projectID, "state", state, "previousState", previousState)
		return
	}
	if s.publisher == nil {
		slog.WarnContext(ctx, "no event publisher configured; query-hour threshold reached but not emailed",
			"projectId", projectID, "state", state)
		return
	}

	nctx, err := s.repo.NotificationContext(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "resolve query-hour notification context", "err", err, "projectId", projectID)
		return
	}

	// ServiceNow added each address only if it ended @wso2.com, then
	// de-duplicated with ArrayUtil.unique(). Same here.
	var to []string
	seen := map[string]bool{}
	for _, addr := range []string{nctx.AccountManagerEmail, nctx.TechnicalOwnerEmail} {
		addr = strings.ToLower(strings.TrimSpace(addr))
		if addr == "" || !strings.HasSuffix(addr, internalEmailDomain) || seen[addr] {
			continue
		}
		seen[addr] = true
		to = append(to, addr)
	}
	if len(to) == 0 {
		// ServiceNow fell back to a single hardcoded address here
		// (kalanad@wso2.com) when the account had no owner. That is one
		// person's inbox standing in for a data problem, and it is not
		// reproduced: the cc groups still receive the notice, so nothing is
		// lost, and the missing owner stays visible instead of being absorbed.
		slog.WarnContext(ctx, "query-hour threshold: no internal owner resolved, sending to the cc groups only",
			"projectId", projectID, "state", state, "accountName", nctx.AccountName)
	}

	cc := append([]string{}, queryHourCcGroups...)
	if state == domain.QueryHourStateExceeded {
		cc = append(cc, queryHourExceededCc)
	}

	accountName := nctx.AccountName
	if accountName == "" {
		accountName = consumption.ProjectKey
	}

	var subject string
	switch state {
	case domain.QueryHourStateExceeded:
		subject = "Query Hour Exceeded in " + accountName
	case domain.QueryHourStateCritical:
		subject = "90% of Query Hours Utilized"
	default:
		subject = "75% of Query Hours Utilized"
	}

	payload, err := json.Marshal(events.QueryHourThresholdReachedPayload{
		ProjectID:          projectID,
		ProjectKey:         consumption.ProjectKey,
		ProjectName:        nctx.ProjectName,
		AccountName:        accountName,
		State:              state,
		PreviousState:      previousState,
		TotalQueryHours:    domain.FormatHoursMinutes(stored.EntitlementMinutes),
		ConsumedHours:      domain.FormatHoursMinutes(stored.ConsumedMinutes),
		RemainingHours:     domain.FormatHoursMinutes(stored.RemainingMinutes),
		EntitlementMinutes: stored.EntitlementMinutes,
		ConsumedMinutes:    stored.ConsumedMinutes,
		RemainingMinutes:   stored.RemainingMinutes,
		PercentConsumed:    stored.PercentConsumed,
		Subject:            subject,
		Recipients:         to,
		CcRecipients:       cc,
	})
	if err != nil {
		slog.ErrorContext(ctx, "marshal query-hour threshold payload", "err", err, "projectId", projectID)
		return
	}
	if err := s.publisher.Publish(ctx, events.TypeQueryHourThresholdReached, projectID, payload); err != nil {
		slog.ErrorContext(ctx, "publish query-hour threshold event", "err", err, "projectId", projectID)
		return
	}
	slog.InfoContext(ctx, "query-hour threshold notice published",
		"projectId", projectID, "state", state, "previousState", previousState,
		"to", len(to), "cc", len(cc))
}
