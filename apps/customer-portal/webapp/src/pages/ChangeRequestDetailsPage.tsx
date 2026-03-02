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

import { useParams, useNavigate } from "react-router";
import { type JSX } from "react";
import {
  Box,
  Button,
  Stack,
  Typography,
  Paper,
  Chip,
  Divider,
  alpha,
  colors,
} from "@wso2/oxygen-ui";
import {
  ArrowLeft,
  Calendar,
  TriangleAlert,
  ChevronRight,
  CircleCheckBig,
  Circle,
  Server,
  Package,
  FileText,
  Users,
  RotateCcw,
  Shield,
  MessageSquare,
  Download,
  ExternalLink,
} from "@wso2/oxygen-ui-icons-react";
import { MOCK_CHANGE_REQUESTS } from "@models/responses";

/**
 * ChangeRequestDetailsPage component to display detailed information about a change request.
 *
 * @returns {JSX.Element} The rendered Change Request Details page.
 */
export default function ChangeRequestDetailsPage(): JSX.Element {
  const navigate = useNavigate();
  const { projectId, changeRequestId } = useParams<{
    projectId: string;
    changeRequestId: string;
  }>();

  // Display mock data for any change request
  const changeRequest =
    MOCK_CHANGE_REQUESTS.find((cr) => cr.id === changeRequestId) ||
    MOCK_CHANGE_REQUESTS[0];

  const getImpactColor = () => {
    switch (changeRequest.impact?.label?.toLowerCase()) {
      case "high":
        return {
          bg: alpha(colors.red[500], 0.1),
          text: colors.red[800],
          border: alpha(colors.red[500], 0.2),
        };
      case "medium":
        return {
          bg: alpha(colors.orange[500], 0.1),
          text: colors.orange[800],
          border: alpha(colors.orange[500], 0.2),
        };
      case "low":
        return {
          bg: alpha(colors.green[500], 0.1),
          text: colors.green[800],
          border: alpha(colors.green[500], 0.2),
        };
      default:
        return {
          bg: alpha(colors.grey[500], 0.1),
          text: colors.grey[800],
          border: alpha(colors.grey[500], 0.2),
        };
    }
  };

  const getStatusColor = () => {
    const label = changeRequest.state?.label?.toLowerCase() || "";
    if (label.includes("scheduled"))
      return {
        bg: alpha(colors.blue[500], 0.1),
        text: colors.blue[800],
        border: alpha(colors.blue[500], 0.2),
      };
    if (label.includes("implement") || label.includes("progress"))
      return {
        bg: alpha(colors.orange[500], 0.1),
        text: colors.orange[800],
        border: alpha(colors.orange[500], 0.2),
      };
    if (label.includes("completed") || label.includes("closed"))
      return {
        bg: alpha(colors.green[500], 0.1),
        text: colors.green[800],
        border: alpha(colors.green[500], 0.2),
      };
    return {
      bg: alpha(colors.grey[500], 0.1),
      text: colors.grey[800],
      border: alpha(colors.grey[500], 0.2),
    };
  };

  const impactColor = getImpactColor();
  const statusColor = getStatusColor();

  // Workflow stages
  const workflowStages = [
    {
      name: "New",
      description: "Change request created",
      date: "Nov 10, 2099",
      completed: true,
    },
    {
      name: "Assess",
      description: "Technical assessment completed",
      date: "Nov 11, 2099",
      completed: true,
    },
    {
      name: "Authorize",
      description: "Internal authorization obtained",
      date: "Nov 12, 2099",
      completed: true,
    },
    {
      name: "Customer Approval",
      description: "Customer approval received",
      date: "Nov 13, 2099",
      completed: true,
    },
    {
      name: "Scheduled",
      description: "Maintenance window scheduled",
      date: "Nov 14, 2099",
      completed: true,
      current: true,
    },
    {
      name: "Implement",
      description: "Change implementation",
      date: null,
      completed: false,
    },
    {
      name: "Review",
      description: "Internal review",
      date: null,
      completed: false,
    },
    {
      name: "Customer Review",
      description: "Customer validation",
      date: null,
      completed: false,
    },
    {
      name: "Closed",
      description: "Change request completed",
      date: null,
      completed: false,
    },
  ];

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
      {/* Back Button */}
      <Button
        startIcon={<ArrowLeft size={16} />}
        onClick={() => navigate(`/${projectId}/support/change-requests`)}
        sx={{ alignSelf: "flex-start" }}
        variant="text"
      >
        Back to Change Requests
      </Button>

      {/* Header Section */}
      <Paper variant="outlined" sx={{ p: 4 }}>
        <Box
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
          }}
        >
          <Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 1 }}>
              <Typography variant="h5" color="text.primary">
                {changeRequest.title}
              </Typography>
              <Chip
                icon={<TriangleAlert size={12} />}
                label="Service Outage"
                size="small"
                sx={{
                  bgcolor: alpha(colors.red[500], 0.1),
                  color: colors.red[800],
                  borderColor: alpha(colors.red[500], 0.2),
                  border: "1px solid",
                  "& .MuiChip-icon": {
                    color: colors.red[800],
                  },
                }}
              />
            </Box>
            <Box
              sx={{
                display: "flex",
                alignItems: "center",
                gap: 1.5,
                fontSize: "0.875rem",
                color: "text.secondary",
              }}
            >
              <Typography variant="body2" fontWeight={600} color="text.primary">
                {changeRequest.number}
              </Typography>
              <Typography variant="body2" color="text.disabled">
                |
              </Typography>
              <Typography variant="body2">
                Service Request: SR-2099-123
              </Typography>
              <Typography variant="body2" color="text.disabled">
                |
              </Typography>
              <Typography variant="body2">
                Created {changeRequest.createdOn}
              </Typography>
            </Box>
          </Box>

          <Box sx={{ display: "flex", gap: 1 }}>
            <Chip
              label={`${changeRequest.impact?.label} Impact`}
              size="small"
              sx={{
                bgcolor: impactColor.bg,
                color: impactColor.text,
                borderColor: impactColor.border,
                border: "1px solid",
              }}
            />
            <Chip
              icon={<Calendar size={12} />}
              label={changeRequest.state?.label || "Not Available"}
              size="small"
              sx={{
                bgcolor: statusColor.bg,
                color: statusColor.text,
                borderColor: statusColor.border,
                border: "1px solid",
                "& .MuiChip-icon": {
                  color: statusColor.text,
                },
              }}
            />
          </Box>
        </Box>
      </Paper>

      {/* Content Section */}
      {/* Workflow Card */}
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 0.5 }}>
            <ChevronRight size={20} color={colors.purple[600]} />
            <Typography variant="h6">Change Request Workflow</Typography>
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Track the progress of this change request through each stage
          </Typography>
        </Box>

        <Box sx={{ px: 3, pb: 3 }}>
          <Stack spacing={0}>
            {workflowStages.map((stage, index) => (
              <Box key={index} sx={{ position: "relative" }}>
                <Box sx={{ display: "flex", gap: 2 }}>
                  <Box
                    sx={{
                      display: "flex",
                      flexDirection: "column",
                      alignItems: "center",
                    }}
                  >
                    <Box
                      sx={{
                        width: 40,
                        height: 40,
                        borderRadius: "50%",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        border: "2px solid",
                        bgcolor: stage.completed
                          ? alpha(colors.green[500], 0.1)
                          : stage.current
                            ? alpha(colors.blue[500], 0.1)
                            : alpha(colors.grey[500], 0.1),
                        borderColor: stage.completed
                          ? colors.green[500]
                          : stage.current
                            ? colors.blue[500]
                            : colors.grey[300],
                      }}
                    >
                      {stage.completed ? (
                        <CircleCheckBig size={20} color={colors.green[600]} />
                      ) : (
                        <Circle
                          size={20}
                          color={
                            stage.current ? colors.blue[600] : colors.grey[400]
                          }
                          fill={stage.current ? colors.blue[600] : "none"}
                        />
                      )}
                    </Box>
                    {index < workflowStages.length - 1 && (
                      <Box
                        sx={{
                          width: 2,
                          height: 64,
                          mt: 0.5,
                          bgcolor: stage.completed
                            ? colors.green[300]
                            : colors.grey[200],
                        }}
                      />
                    )}
                  </Box>
                  <Box
                    sx={{
                      flex: 1,
                      pb: index < workflowStages.length - 1 ? 2 : 0,
                    }}
                  >
                    <Box
                      sx={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "flex-start",
                      }}
                    >
                      <Box>
                        <Box
                          sx={{
                            display: "flex",
                            alignItems: "center",
                            gap: 1,
                            mb: 0.5,
                          }}
                        >
                          <Typography
                            variant="body2"
                            fontWeight={stage.current ? 600 : 500}
                            color={
                              stage.current ? colors.blue[900] : "text.primary"
                            }
                          >
                            {stage.name}
                          </Typography>
                          {stage.current && (
                            <Chip
                              label="Current Stage"
                              size="small"
                              sx={{
                                bgcolor: alpha(colors.blue[500], 0.1),
                                color: colors.blue[800],
                                borderColor: alpha(colors.blue[500], 0.2),
                                border: "1px solid",
                                height: 20,
                                fontSize: "0.7rem",
                              }}
                            />
                          )}
                        </Box>
                        <Typography
                          variant="caption"
                          color={
                            stage.completed || stage.current
                              ? "text.primary"
                              : "text.disabled"
                          }
                        >
                          {stage.description}
                        </Typography>
                      </Box>
                      {stage.date && (
                        <Typography variant="caption" color="text.secondary">
                          {stage.date}
                        </Typography>
                      )}
                    </Box>
                  </Box>
                </Box>
              </Box>
            ))}
          </Stack>
        </Box>
      </Paper>

      {/* Deployment & Component Card */}
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Server size={20} color={colors.grey[600]} />
            <Typography variant="h6">Deployment & Component</Typography>
          </Box>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Box sx={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 3 }}>
            <Box>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                Deployment
              </Typography>
              <Typography variant="body2">
                {changeRequest.deployment?.label}
              </Typography>
            </Box>
            <Box>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                Component
              </Typography>
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <Package size={16} color={colors.grey[400]} />
                <Typography variant="body2">WSO2 API Manager</Typography>
              </Box>
            </Box>
          </Box>
        </Box>
      </Paper>

      {/* Change Description Card */}
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <FileText size={20} color={colors.grey[600]} />
            <Typography variant="h6">Change Description</Typography>
          </Box>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Stack spacing={3}>
            <Box>
              <Typography
                variant="body2"
                color="text.primary"
                fontWeight={500}
                sx={{ mb: 1 }}
              >
                What changes are being made?
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {changeRequest.title || "No description available"}
              </Typography>
            </Box>

            <Divider />

            <Box>
              <Typography
                variant="body2"
                color="text.primary"
                fontWeight={500}
                sx={{ mb: 1 }}
              >
                Why is this change needed?
              </Typography>
              <Typography variant="body2" color="text.secondary">
                This upgrade is needed to address critical security
                vulnerabilities and to gain access to new features including
                improved rate limiting, enhanced analytics, and better OAuth 2.1
                support.
              </Typography>
            </Box>

            <Divider />

            <Box>
              <Typography
                variant="body2"
                color="text.primary"
                fontWeight={500}
                sx={{ mb: 1 }}
              >
                Expected Outcome
              </Typography>
              <Typography variant="body2" color="text.secondary">
                After the upgrade, all APIs will be running on the latest
                version with improved security posture, enhanced performance,
                and access to new features. All existing APIs and subscriptions
                will continue to function without requiring changes.
              </Typography>
            </Box>
          </Stack>
        </Box>
      </Paper>

      {/* Impact Analysis Card */}
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <TriangleAlert size={20} color={colors.red[600]} />
            <Typography variant="h6">Impact Analysis</Typography>
          </Box>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Stack spacing={3}>
            <Box>
              <Typography
                variant="body2"
                color="text.primary"
                fontWeight={500}
                sx={{ mb: 1 }}
              >
                Impact Description
              </Typography>
              <Typography variant="body2" color="text.secondary">
                API Gateway will be unavailable during upgrade. All API traffic
                will be affected. Approximately 10,000 active API calls per
                minute will be disrupted during the maintenance window.
              </Typography>
            </Box>

            <Divider />

            <Box>
              <Typography
                variant="body2"
                color="text.primary"
                fontWeight={500}
                sx={{ mb: 2 }}
              >
                Service Outage Details
              </Typography>
              <Paper
                sx={{
                  bgcolor: alpha(colors.red[500], 0.05),
                  border: 1,
                  borderColor: alpha(colors.red[500], 0.2),
                  p: 2,
                }}
              >
                <Box sx={{ display: "flex", gap: 1.5 }}>
                  <TriangleAlert
                    size={20}
                    color={colors.red[600]}
                    style={{ marginTop: 2 }}
                  />
                  <Typography variant="body2" color="text.primary">
                    Complete service outage for 4 hours during the upgrade
                    process. The API Gateway cluster will be taken offline for
                    the upgrade. No API traffic will be processed during this
                    time.
                  </Typography>
                </Box>
              </Paper>
            </Box>
          </Stack>
        </Box>
      </Paper>

      {/* Communication Plan Card */}
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Users size={20} color={colors.grey[600]} />
            <Typography variant="h6">Communication Plan</Typography>
          </Box>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Typography variant="body2" color="text.secondary">
            Email notification to all API consumers 7 days before the change.
            Follow-up reminder 24 hours before. SMS alerts to on-call engineers
            1 hour before start. Post-maintenance status update within 1 hour of
            completion.
          </Typography>
        </Box>
      </Paper>
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <RotateCcw size={20} color={colors.grey[600]} />
            <Typography variant="h6">Rollback Plan</Typography>
          </Box>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Typography variant="body2" color="text.secondary">
            If critical issues are encountered, rollback to previous version
            using the pre-upgrade snapshot. Database rollback scripts are
            prepared. Estimated rollback time: 1 hour. Rollback decision point:
            2 hours into the maintenance window.
          </Typography>
        </Box>
      </Paper>
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Shield size={20} color={colors.grey[600]} />
            <Typography variant="h6">Test Plan</Typography>
          </Box>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Typography variant="body2" color="text.secondary">
            Pre-production testing completed in staging environment (Nov 20-25,
            2099). Smoke tests include: API invocation, token generation,
            subscription workflows, rate limiting, and analytics collection.
            Post-upgrade validation includes all critical API endpoints.
          </Typography>
        </Box>
      </Paper>

      {/* Approval Information Card */}
      <Paper variant="outlined">
        <Box sx={{ px: 3, pt: 3, pb: 3 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Approval Information
          </Typography>

          <Stack spacing={1.5}>
            <Box sx={{ display: "flex", justifyContent: "space-between" }}>
              <Typography variant="body2" color="text.secondary">
                Created By
              </Typography>
              <Typography variant="body2">Not Available</Typography>
            </Box>
            <Box sx={{ display: "flex", justifyContent: "space-between" }}>
              <Typography variant="body2" color="text.secondary">
                Created Date
              </Typography>
              <Typography variant="body2">{changeRequest.createdOn}</Typography>
            </Box>

            <Divider />

            <Box
              sx={{
                display: "flex",
                justifyContent: "space-between",
                opacity: 0.6,
                pointerEvents: "none",
              }}
            >
              <Typography variant="body2" color="text.secondary">
                Approved By
              </Typography>
              <Typography variant="body2" color="text.disabled">
                Not available
              </Typography>
            </Box>
            <Box
              sx={{
                display: "flex",
                justifyContent: "space-between",
                opacity: 0.6,
                pointerEvents: "none",
              }}
            >
              <Typography variant="body2" color="text.secondary">
                Approved Date
              </Typography>
              <Typography variant="body2" color="text.disabled">
                Not available
              </Typography>
            </Box>
          </Stack>
        </Box>
      </Paper>

      {/* Notes & Comments Card */}
      <Paper variant="outlined" sx={{ opacity: 0.6, pointerEvents: "none" }}>
        <Box sx={{ px: 3, pt: 3 }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 0.5 }}>
            <MessageSquare size={20} />
            <Typography variant="h6">Notes & Comments</Typography>
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Track updates and communications related to this change request
          </Typography>
        </Box>

        <Box sx={{ px: 3, py: 3 }}>
          <Typography
            variant="body2"
            color="text.disabled"
            sx={{ textAlign: "center", py: 3 }}
          >
            No notes or comments available
          </Typography>
        </Box>
      </Paper>
      {/* Action Buttons */}
      <Box sx={{ display: "flex", gap: 2 }}>
        <Button
          variant="outlined"
          startIcon={<Download size={18} />}
          sx={{ flex: 1 }}
        >
          Download Change Request PDF
        </Button>
        <Button
          variant="outlined"
          startIcon={<ExternalLink size={18} />}
          sx={{ flex: 1 }}
        >
          View Related Service Request
        </Button>
      </Box>
    </Box>
  );
}
