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

import { Grid } from "@wso2/oxygen-ui";
import { CircleAlert, Shield, FileCheck } from "@wso2/oxygen-ui-icons-react";
import { useParams } from "react-router";
import { useMemo, type JSX } from "react";
import { StatCard } from "@components/dashboard/stats/StatCard";
import { usePostProductVulnerabilitiesSearch } from "@api/usePostProductVulnerabilitiesSearch";
import { useGetProjectCasesStats } from "@api/useGetProjectCasesStats";
import { CaseType } from "@constants/supportConstants";

/**
 * SecurityStats component displays security-related statistics cards.
 *
 * @returns {JSX.Element} The rendered SecurityStats component.
 */
export default function SecurityStats(): JSX.Element {
  const { projectId } = useParams<{ projectId: string }>();

  const {
    data: vulnerabilitiesData,
    isLoading: isVulnerabilitiesLoading,
    isError: isVulnerabilitiesError,
  } = usePostProductVulnerabilitiesSearch({
    pagination: {
      offset: 0,
      limit: 10,
    },
  });

  const {
    data: securityReportStats,
    isLoading: isSecurityReportLoading,
    isError: isSecurityReportError,
  } = useGetProjectCasesStats(projectId || "", {
    caseTypes: [CaseType.SECURITY_REPORT_ANALYSIS],
    enabled: !!projectId,
  });

  const stats = useMemo(
    () => [
      {
        id: "totalVulnerabilities",
        label: "Total Vulnerabilities",
        value: vulnerabilitiesData?.totalRecords ?? 0,
        icon: <CircleAlert size={20} />,
        iconColor: "error" as const,
        tooltipText: "Total number of vulnerabilities detected",
        isLoading: isVulnerabilitiesLoading,
        isError: isVulnerabilitiesError,
      },
      {
        id: "activeSecurityReports",
        label: "Active Security Reports",
        value: securityReportStats?.activeCount ?? 0,
        icon: <Shield size={20} />,
        iconColor: "warning" as const,
        tooltipText: "Number of active security report analysis cases",
        isLoading: isSecurityReportLoading,
        isError: isSecurityReportError,
      },
      {
        id: "resolvedSecurityReports",
        label: "Resolved Security Reports (Last 30d)",
        value: securityReportStats?.resolvedCases?.pastThirtyDays ?? 0,
        icon: <FileCheck size={20} />,
        iconColor: "success" as const,
        tooltipText:
          "Number of security reports resolved in the past 30 days",
        isLoading: isSecurityReportLoading,
        isError: isSecurityReportError,
      },
    ],
    [
      vulnerabilitiesData,
      isVulnerabilitiesLoading,
      isVulnerabilitiesError,
      securityReportStats,
      isSecurityReportLoading,
      isSecurityReportError,
    ],
  );

  return (
    <Grid container spacing={2} sx={{ mb: 3 }}>
      {stats.map((stat) => (
        <Grid key={stat.id} size={{ xs: 12, sm: 6, md: 4 }}>
          <StatCard
            label={stat.label}
            value={stat.value}
            icon={stat.icon}
            iconColor={stat.iconColor}
            tooltipText={stat.tooltipText}
            showTrend={false}
            isLoading={stat.isLoading}
            isError={stat.isError}
            isTrendError={false}
          />
        </Grid>
      ))}
    </Grid>
  );
}
