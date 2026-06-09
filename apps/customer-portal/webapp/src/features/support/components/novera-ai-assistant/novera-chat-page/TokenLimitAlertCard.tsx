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

import type { TokenLimitAlertCardProps } from "@features/support/types/supportComponents";
import {
  Alert,
  Box,
  Button,
  Collapse,
  Stack,
  TextField,
  Typography,
} from "@wso2/oxygen-ui";
import { type JSX, useState } from "react";

/**
 * Collapsible alert card shown when a token limit is reached.
 * Collapsed by default; expands to show a reason form for requesting an increase.
 */
export default function TokenLimitAlertCard({
  scope,
  message,
  acknowledged,
  onRequestIncrease,
}: TokenLimitAlertCardProps): JSX.Element {
  const [expanded, setExpanded] = useState(false);
  const [reason, setReason] = useState("");
  const [requestedLimit, setRequestedLimit] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const title =
    scope === "account"
      ? "Monthly Token Limit Reached"
      : "Daily Session Limit Reached";

  const handleSubmit = () => {
    const trimmed = reason.trim();
    if (!trimmed) return;
    setIsSubmitting(true);
    const parsed = requestedLimit.trim() ? parseInt(requestedLimit, 10) : undefined;
    onRequestIncrease(trimmed, Number.isFinite(parsed) ? parsed : undefined);
  };

  return (
    <Alert
      severity="warning"
      sx={{ maxWidth: 520 }}
      action={
        !acknowledged ? (
          <Button
            size="small"
            color="inherit"
            onClick={() => setExpanded((prev) => !prev)}
            sx={{ whiteSpace: "nowrap", alignSelf: "flex-start", mt: 0.25 }}
          >
            {expanded ? "Hide" : "Request Increase"}
          </Button>
        ) : undefined
      }
    >
      <Typography variant="body2" fontWeight={600}>
        {title}
      </Typography>
      <Typography variant="caption" color="text.secondary">
        {message}
      </Typography>

      <Collapse in={expanded && !acknowledged}>
        <Stack spacing={1.5} sx={{ mt: 1.5 }}>
          <TextField
            multiline
            minRows={2}
            maxRows={4}
            size="small"
            placeholder="Briefly describe why you need a higher limit..."
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            disabled={isSubmitting}
            fullWidth
          />
          <TextField
            type="number"
            size="small"
            label="Requested token limit (optional)"
            placeholder="e.g. 200000"
            value={requestedLimit}
            onChange={(e) => setRequestedLimit(e.target.value)}
            disabled={isSubmitting}
            slotProps={{ htmlInput: { min: 1 } }}
            fullWidth
          />
          <Box>
            <Button
              variant="contained"
              size="small"
              color="warning"
              disabled={!reason.trim() || isSubmitting}
              onClick={handleSubmit}
            >
              {isSubmitting ? "Submitting..." : "Submit Request"}
            </Button>
          </Box>
        </Stack>
      </Collapse>

      {acknowledged && (
        <Typography variant="caption" color="success.main" sx={{ mt: 0.5, display: "block" }}>
          ✓ Request submitted. The support team will review it.
        </Typography>
      )}
    </Alert>
  );
}
