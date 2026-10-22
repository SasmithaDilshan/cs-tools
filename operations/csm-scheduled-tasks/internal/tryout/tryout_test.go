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

package tryout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wso2-open-operations/cs-tools/operations/csm-scheduled-tasks/internal/ledger"
	"github.com/wso2-open-operations/cs-tools/operations/csm-scheduled-tasks/internal/registry"
	"github.com/wso2-open-operations/cs-tools/operations/csm-scheduled-tasks/internal/tryout"
)

type fakeLedger struct {
	claim       ledger.Claim
	attemptName string
	completed   int
	failed      int
	failRetry   time.Time
}

func (f *fakeLedger) Attempt(_ context.Context, taskName string, _ time.Time, _ time.Duration) (ledger.Claim, error) {
	f.attemptName = taskName
	return f.claim, nil
}
func (f *fakeLedger) Complete(context.Context, string, int) error { f.completed++; return nil }
func (f *fakeLedger) Fail(_ context.Context, _ string, _ int, _ string, next time.Time) error {
	f.failed++
	f.failRetry = next
	return nil
}

func tasks(calls *int, err error) []registry.Task {
	return []registry.Task{
		{Name: "other", Handler: func(context.Context) error { return nil }},
		{Name: "report", Handler: func(context.Context) error { *calls++; return err }},
	}
}

func TestRunPerformsTheTryoutOncePerRunID(t *testing.T) {
	calls := 0
	led := &fakeLedger{claim: ledger.Claim{Allowed: true, Run: ledger.Run{ID: "r1", AttemptCount: 1}}}
	out, err := tryout.Run(context.Background(), led, tasks(&calls, nil), "report", "2026-10-22", []string{"me@example.com"}, true, time.Hour)
	if err != nil || out != tryout.Ran {
		t.Fatalf("Run = %v, %v; want Ran", out, err)
	}
	if calls != 1 || led.completed != 1 || led.failed != 0 {
		t.Errorf("handler calls %d, completed %d, failed %d; want 1/1/0", calls, led.completed, led.failed)
	}
	if led.attemptName != "tryout:report@2026-10-22" {
		t.Errorf("ledger task name = %q", led.attemptName)
	}
}

func TestRunDefaultsTheRunID(t *testing.T) {
	calls := 0
	led := &fakeLedger{claim: ledger.Claim{Allowed: true}}
	if _, err := tryout.Run(context.Background(), led, tasks(&calls, nil), "report", "  ", []string{"me@example.com"}, true, time.Hour); err != nil {
		t.Fatal(err)
	}
	if led.attemptName != "tryout:report@1" {
		t.Errorf("ledger task name = %q, want the default run id", led.attemptName)
	}
}

func TestRunSkipsWhenAlreadyDoneOrClaimedElsewhere(t *testing.T) {
	calls := 0
	done := time.Now()
	led := &fakeLedger{claim: ledger.Claim{Allowed: false, Run: ledger.Run{SucceededOn: &done}}}
	out, err := tryout.Run(context.Background(), led, tasks(&calls, nil), "report", "1", []string{"me@example.com"}, true, time.Hour)
	if err != nil || out != tryout.AlreadyDone || calls != 0 {
		t.Errorf("already done: out=%v err=%v calls=%d", out, err, calls)
	}

	msg := "boom"
	led = &fakeLedger{claim: ledger.Claim{Allowed: false, Run: ledger.Run{LastError: &msg}}}
	out, err = tryout.Run(context.Background(), led, tasks(&calls, nil), "report", "1", []string{"me@example.com"}, true, time.Hour)
	if err != nil || out != tryout.AlreadyDone || calls != 0 {
		t.Errorf("already failed: out=%v err=%v calls=%d; a failed tryout is not retried", out, err, calls)
	}

	led = &fakeLedger{claim: ledger.Claim{Allowed: false}}
	out, err = tryout.Run(context.Background(), led, tasks(&calls, nil), "report", "1", []string{"me@example.com"}, true, time.Hour)
	if err != nil || out != tryout.ClaimedElsewhere || calls != 0 {
		t.Errorf("claimed elsewhere: out=%v err=%v calls=%d", out, err, calls)
	}
}

func TestRunRefusesWithoutDebugRecipientsOrUnknownTask(t *testing.T) {
	calls := 0
	led := &fakeLedger{claim: ledger.Claim{Allowed: true}}
	if _, err := tryout.Run(context.Background(), led, tasks(&calls, nil), "report", "1", nil, true, time.Hour); !errors.Is(err, tryout.ErrRefused) {
		t.Errorf("no debug recipients: err = %v", err)
	}
	if _, err := tryout.Run(context.Background(), led, tasks(&calls, nil), "nope", "1", []string{"me@example.com"}, true, time.Hour); !errors.Is(err, tryout.ErrRefused) {
		t.Errorf("unknown task: err = %v", err)
	}
	if calls != 0 || led.attemptName != "" {
		t.Errorf("a refused tryout must touch neither the handler nor the ledger (calls %d, claim %q)", calls, led.attemptName)
	}
}

func TestRunRecordsAHandlerFailureWithoutRetry(t *testing.T) {
	calls := 0
	boom := errors.New("boom")
	led := &fakeLedger{claim: ledger.Claim{Allowed: true, Run: ledger.Run{ID: "r1", AttemptCount: 1}}}
	_, err := tryout.Run(context.Background(), led, tasks(&calls, boom), "report", "1", []string{"me@example.com"}, true, time.Hour)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the handler's", err)
	}
	if led.failed != 1 || led.completed != 0 || led.failRetry.Year() != 9999 {
		t.Errorf("failed %d, completed %d, retry %v; want 1/0/never", led.failed, led.completed, led.failRetry)
	}
}
