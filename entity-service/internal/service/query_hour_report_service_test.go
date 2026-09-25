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
	"testing"

	"github.com/wso2-open-operations/cs-tools/entity-service/internal/domain"
)

// row is a terse constructor for the flat rows the repository returns.
func row(account, opp string, entitlement int, project string, consumed int) domain.QueryHoursReportRow {
	return domain.QueryHoursReportRow{
		AccountID:          account,
		AccountName:        account,
		AccountSFID:        "sf-" + account,
		OpportunityID:      opp,
		OpportunityName:    opp,
		EntitlementMinutes: entitlement,
		ProjectID:          project,
		ProjectName:        project,
		ProjectKey:         project,
		ConsumedMinutes:    consumed,
	}
}

// The threshold is an absolute floor of 600 minutes, NOT a percentage. These
// are the cases that a percentage rule would get wrong, and they are the
// whole reason the report cannot reuse QueryHourStateFor.
func TestBuildGroup_ThresholdIsAnAbsoluteFloorNotAPercentage(t *testing.T) {
	tests := []struct {
		name          string
		entitlement   int
		consumed      int
		exceeded      bool
		goingToExceed bool
	}{
		{"exhausted past zero is exceeded", 600, 700, true, false},
		{"exactly zero remaining is exceeded only when negative", 600, 600, false, true},
		{"one minute left is going to exceed", 600, 599, false, true},
		{"exactly the floor is not yet going to exceed", 1200, 600, false, false},
		{"one minute under the floor is going to exceed", 1200, 601, false, true},
		// 95% consumed, far past any percentage threshold, but 300 minutes
		// remain — a percentage rule would flag it, and so does this, but for
		// a different reason. Kept so the two rules are not quietly merged.
		{"ninety-five percent used with 300 minutes left", 6000, 5700, false, true},
		// 0% consumed. A percentage rule would never flag this; the floor
		// does, because sixty minutes cannot absorb another billable hour.
		{"one hour entitlement untouched is still going to exceed", 60, 0, false, true},
		// No entitlement at all. ServiceNow reports these and so do we.
		{"no entitlement at all is going to exceed", 0, 0, false, true},
		{"comfortable headroom is in neither table", 6000, 100, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := buildGroup([]domain.QueryHoursReportRow{
				row("a", "o1", tc.entitlement, "p1", tc.consumed),
			})
			if g.Exceeded != tc.exceeded || g.GoingToExceed != tc.goingToExceed {
				t.Fatalf("entitlement=%d consumed=%d remaining=%d: got exceeded=%v goingToExceed=%v, want %v/%v",
					tc.entitlement, tc.consumed, g.RemainingMinutes,
					g.Exceeded, g.GoingToExceed, tc.exceeded, tc.goingToExceed)
			}
		})
	}
}

// Two opportunities funding one project are one component, and the project's
// consumption is counted ONCE. Counting it per row would double it and put a
// healthy account in the exceeded table.
func TestComponentsOf_OpportunitiesSharingAProjectMergeAndConsumptionCountsOnce(t *testing.T) {
	rows := []domain.QueryHoursReportRow{
		row("acct", "o1", 6000, "shared", 4000),
		row("acct", "o2", 6000, "shared", 4000),
	}
	components := componentsOf(rows)
	if len(components) != 1 {
		t.Fatalf("expected the two opportunities to merge into 1 component, got %d", len(components))
	}

	g := buildGroup(components[0])
	if g.EntitlementMinutes != 12000 {
		t.Errorf("entitlement should count each opportunity once: got %d, want 12000", g.EntitlementMinutes)
	}
	if g.ConsumedMinutes != 4000 {
		t.Errorf("consumption should count the shared project once: got %d, want 4000", g.ConsumedMinutes)
	}
	if g.RemainingMinutes != 8000 {
		t.Errorf("remaining: got %d, want 8000", g.RemainingMinutes)
	}
	if g.Exceeded || g.GoingToExceed {
		t.Errorf("a group with 8000 minutes left belongs in neither table")
	}

	// The repeated row is still rendered, but marked so the template can grey
	// it out the way ServiceNow does.
	if !g.Opportunities[1].Projects[0].Duplicate {
		t.Error("the second opportunity's row for the shared project should be marked Duplicate")
	}
	if g.Opportunities[0].Projects[0].Duplicate {
		t.Error("the first occurrence of a project must not be marked Duplicate")
	}
}

// Transitivity: o1 funds p1, o2 funds p1 and p2, o3 funds p2. All three are
// one component even though o1 and o3 share no project directly. ServiceNow's
// pairwise union check gives up on exactly this shape and drops the account.
func TestComponentsOf_MergesTransitively(t *testing.T) {
	rows := []domain.QueryHoursReportRow{
		row("acct", "o1", 600, "p1", 0),
		row("acct", "o2", 600, "p1", 0),
		row("acct", "o2", 600, "p2", 0),
		row("acct", "o3", 600, "p2", 0),
	}
	components := componentsOf(rows)
	if len(components) != 1 {
		t.Fatalf("o1 and o3 are linked through o2; expected 1 component, got %d", len(components))
	}
	g := buildGroup(components[0])
	if len(g.Opportunities) != 3 {
		t.Fatalf("expected all 3 opportunities in the component, got %d", len(g.Opportunities))
	}
	if g.EntitlementMinutes != 1800 {
		t.Errorf("each opportunity counted once: got %d, want 1800", g.EntitlementMinutes)
	}
}

// Unrelated opportunities must NOT merge — otherwise one account's healthy
// entitlement would mask another's overage.
func TestComponentsOf_KeepsUnrelatedOpportunitiesApart(t *testing.T) {
	rows := []domain.QueryHoursReportRow{
		row("acct", "o1", 6000, "p1", 0),
		row("acct", "o2", 60, "p2", 600),
	}
	components := componentsOf(rows)
	if len(components) != 2 {
		t.Fatalf("expected 2 independent components, got %d", len(components))
	}
}

// The flags are computed on the MERGED totals. ServiceNow computes them
// per-opportunity before merging and never recomputes, which is how an
// account shows a large remaining balance and still sits in the
// going-to-exceed table.
func TestBuildGroup_ThresholdsUseMergedTotalsNotPreMergeOnes(t *testing.T) {
	// Pre-merge, o2 has no entitlement and zero remaining, which on its own
	// would be flagged. Merged, the component has 6000 minutes spare.
	g := buildGroup([]domain.QueryHoursReportRow{
		row("acct", "o1", 6000, "shared", 0),
		row("acct", "o2", 0, "shared", 0),
	})
	if g.RemainingMinutes != 6000 {
		t.Fatalf("merged remaining: got %d, want 6000", g.RemainingMinutes)
	}
	if g.GoingToExceed {
		t.Error("a merged group with 100 hours spare must not be reported as going to exceed")
	}
}

// An account lands in a table if ANY of its groups qualifies, so the two
// tables overlap by design.
func TestBuildAccount_FlagsAreAnOrAcrossGroupsSoTablesOverlap(t *testing.T) {
	account := buildAccount([]domain.QueryHoursReportRow{
		row("acct", "exceeded", 600, "p1", 900),
		row("acct", "nearly", 1200, "p2", 700),
	})
	if !account.Exceeded {
		t.Error("an account with one exceeded group must appear in the exceeded table")
	}
	if !account.GoingToExceed {
		t.Error("the same account with one nearly-exhausted group must ALSO appear in the going-to-exceed table")
	}
	if account.RowCount != 2 {
		t.Errorf("RowCount: got %d, want 2", account.RowCount)
	}
}

func TestGroupRowsByAccount_SplitsOnAccountBoundaries(t *testing.T) {
	groups := groupRowsByAccount([]domain.QueryHoursReportRow{
		row("a", "o1", 60, "p1", 0),
		row("a", "o2", 60, "p2", 0),
		row("b", "o3", 60, "p3", 0),
	})
	if len(groups) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(groups))
	}
	if len(groups[0]) != 2 || len(groups[1]) != 1 {
		t.Fatalf("unexpected split: %d and %d rows", len(groups[0]), len(groups[1]))
	}
}

// The unmatched-line count travels on every row sharing an opportunity, so it
// must be counted once per opportunity rather than summed across rows.
func TestUnmatchedLines_CountsEachOpportunityOnce(t *testing.T) {
	r1 := row("acct", "o1", 600, "p1", 0)
	r1.UnmatchedLineCount = 2
	r2 := row("acct", "o1", 600, "p2", 0)
	r2.UnmatchedLineCount = 2
	r3 := row("acct", "o2", 600, "p3", 0)
	r3.UnmatchedLineCount = 1

	if got := unmatchedLines([]domain.QueryHoursReportRow{r1, r2, r3}); got != 3 {
		t.Fatalf("got %d, want 3 (2 from o1 counted once, 1 from o2)", got)
	}
}
