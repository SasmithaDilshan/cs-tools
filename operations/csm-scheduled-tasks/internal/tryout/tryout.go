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

// Package tryout runs one registered task once, on demand, outside its
// schedule: the way a report is checked in a real deployment before its
// audience sees it (see cmd/server/main.go's TRYOUT_TASK).
//
// A tryout is recorded in the same ledger the engine uses, under its own
// task name, so it runs exactly once per run id however many processes
// start while TRYOUT_TASK is set -- Choreo may start more than one
// instance per trigger, and the trigger keeps firing on its cadence. The
// first instance to claim the run performs it; every other start, and
// every later tick, sees the claim and carries on with the normal tick,
// so no other task stalls. To run the tryout again, set a new run id.
package tryout

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wso2-open-operations/cs-tools/operations/csm-scheduled-tasks/internal/ledger"
	"github.com/wso2-open-operations/cs-tools/operations/csm-scheduled-tasks/internal/registry"
)

// Ledger is the subset of *ledger.Client a tryout depends on.
type Ledger interface {
	Attempt(ctx context.Context, taskName string, periodKey time.Time, staleClaimAfter time.Duration) (ledger.Claim, error)
	Complete(ctx context.Context, id string, attemptCount int) error
	Fail(ctx context.Context, id string, attemptCount int, errMsg string, nextRetryOn time.Time) error
}

// periodKey is the one period every tryout is recorded under: the run id
// goes in the ledger task name, so the period itself never varies.
var periodKey = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// never is the retry instant recorded for a failed tryout: a tryout is
// never retried on its own, only re-run under a new run id.
var never = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

// Outcome says what Run did.
type Outcome int

const (
	// Ran: this process performed the tryout, and the handler succeeded.
	Ran Outcome = iota
	// AlreadyDone: a tryout with this run id has already succeeded (or
	// failed); the caller should carry on with the normal tick.
	AlreadyDone
	// ClaimedElsewhere: another live instance is performing this tryout
	// right now; the caller should carry on with the normal tick.
	ClaimedElsewhere
)

// ErrRefused is returned when the tryout must not run at all: the task is
// unknown, or it has no debug recipients.
var ErrRefused = errors.New("tryout refused")

// LedgerTaskName is the ledger task name a tryout is recorded under.
func LedgerTaskName(task, runID string) string {
	return "tryout:" + task + "@" + runID
}

// Run performs the tryout of the task named name once per runID.
//
// It is refused unless the task has debug recipients, since the only thing
// a tryout must never do is send a real report to real people on
// somebody's say-so; and it warns, rather than refuses, when emailsEnabled
// is false, because every report task then succeeds without sending
// anything, which is a confusing way to learn nothing.
//
// A handler failure is recorded in the ledger (never retried) and returned
// as the error, so the caller can exit non-zero and the run shows as
// failed where the process was started.
func Run(ctx context.Context, led Ledger, tasks []registry.Task, name, runID string, debugRecipients []string, emailsEnabled bool, staleClaimAfter time.Duration) (Outcome, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = "1"
	}
	if len(debugRecipients) == 0 {
		return 0, fmt.Errorf("%w: task %q has no \"debug\" list in SUB_CRON_RECIPIENTS, and a tryout sends to the debug recipients only", ErrRefused, name)
	}
	var names []string
	var task *registry.Task
	for i := range tasks {
		names = append(names, tasks[i].Name)
		if tasks[i].Name == name {
			task = &tasks[i]
		}
	}
	if task == nil {
		return 0, fmt.Errorf("%w: TRYOUT_TASK %q names no registered task (registered: %s)", ErrRefused, name, strings.Join(names, ", "))
	}
	if !emailsEnabled {
		slog.Warn("tryout with ALERTS_ENABLED=false: report tasks send nothing in that mode, so expect no email", "task", name)
	}

	ledgerName := LedgerTaskName(name, runID)
	claim, err := led.Attempt(ctx, ledgerName, periodKey, staleClaimAfter)
	if err != nil {
		return 0, fmt.Errorf("tryout: claim %s: %w", ledgerName, err)
	}
	if !claim.Allowed {
		if claim.Run.SucceededOn != nil || claim.Run.LastError != nil {
			slog.Info("tryout already performed for this run id; carrying on with the normal tick (set a new TRYOUT_RUN_ID to run it again, or remove TRYOUT_TASK)",
				"task", name, "runId", runID, "succeededOn", claim.Run.SucceededOn)
			return AlreadyDone, nil
		}
		slog.Info("tryout is being performed by another instance; carrying on with the normal tick", "task", name, "runId", runID)
		return ClaimedElsewhere, nil
	}

	slog.Warn("tryout: running the task once, outside its schedule",
		"task", name, "runId", runID, "debugRecipients", len(debugRecipients), "attempt", claim.Run.AttemptCount)
	start := time.Now()
	if handlerErr := task.Handler(ctx); handlerErr != nil {
		if err := led.Fail(ctx, claim.Run.ID, claim.Run.AttemptCount, handlerErr.Error(), never); err != nil {
			slog.Error("tryout: record failure", "task", name, "runId", runID, "err", err)
		}
		return 0, fmt.Errorf("tryout %s failed after %s: %w", name, time.Since(start).Round(time.Millisecond), handlerErr)
	}
	if err := led.Complete(ctx, claim.Run.ID, claim.Run.AttemptCount); err != nil {
		// The report went out; a lost completion only risks a second
		// tryout on the next start. Say so loudly rather than fail.
		slog.Error("tryout: record completion; the next start may run it again", "task", name, "runId", runID, "err", err)
	}
	slog.Info("tryout complete", "task", name, "runId", runID, "elapsed", time.Since(start).String())
	return Ran, nil
}
