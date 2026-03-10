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

import { Box, Paper, Typography, IconButton } from "@wso2/oxygen-ui";
import { Bot, User, ThumbsUp, ThumbsDown } from "@wso2/oxygen-ui-icons-react";
import ReactMarkdown from "react-markdown";
import { type JSX, useState } from "react";
import type { Message } from "@pages/NoveraChatPage";
import RecommendationsCard from "@components/support/novera-ai-assistant/novera-chat-page/RecommendationsCard";

/** Safe URL protocols for markdown links. Blocks javascript:, data:, etc. */
const SAFE_PROTOCOLS = ["http:", "https:"];

function isSafeHref(href: string | undefined): href is string {
  if (!href || typeof href !== "string") return false;
  try {
    const parsed = new URL(href, "https://invalid.invalid");
    return SAFE_PROTOCOLS.includes(parsed.protocol);
  } catch {
    return false;
  }
}

interface ChatMessageBubbleProps {
  message: Message;
  onCreateCase?: () => void;
  onThumbsUp?: (messageId: string) => void;
  onThumbsDown?: (messageId: string) => void;
}

/** Typography mapping for markdown elements (bot messages). */
const markdownComponents: React.ComponentProps<
  typeof ReactMarkdown
>["components"] = {
  h1: ({ children }) => (
    <Typography
      variant="h6"
      component="h1"
      sx={{ mt: 2, mb: 1, fontWeight: 600 }}
    >
      {children}
    </Typography>
  ),
  h2: ({ children }) => (
    <Typography
      variant="subtitle1"
      component="h2"
      sx={{ mt: 2, mb: 1, fontWeight: 600 }}
    >
      {children}
    </Typography>
  ),
  h3: ({ children }) => (
    <Typography
      variant="subtitle2"
      component="h3"
      sx={{ mt: 1.5, mb: 0.5, fontWeight: 600 }}
    >
      {children}
    </Typography>
  ),
  p: ({ children }) => (
    <Typography variant="body2" component="p" sx={{ mb: 1, lineHeight: 1.6 }}>
      {children}
    </Typography>
  ),
  ul: ({ children }) => (
    <Box component="ul" sx={{ m: 0, pl: 2.5, mb: 1 }}>
      {children}
    </Box>
  ),
  ol: ({ children }) => (
    <Box component="ol" sx={{ m: 0, pl: 2.5, mb: 1 }}>
      {children}
    </Box>
  ),
  li: ({ children }) => (
    <Typography
      variant="body2"
      component="li"
      sx={{ mb: 0.5, lineHeight: 1.6 }}
    >
      {children}
    </Typography>
  ),
  strong: ({ children }) => (
    <Box component="strong" sx={{ fontWeight: 600 }}>
      {children}
    </Box>
  ),
  a: ({ href, children }) =>
    isSafeHref(href) ? (
      <Typography
        component="a"
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        variant="body2"
        sx={{ color: "primary.main", textDecoration: "underline" }}
      >
        {children}
      </Typography>
    ) : (
      <Typography variant="body2" component="span">
        {children}
      </Typography>
    ),
};

/**
 * Renders a single chat message bubble.
 *
 * Supports both user and bot messages with appropriate styling,
 * avatar display, markdown formatting for bot messages,
 * optional Create Case action, and displays error text when present.
 *
 * @returns The ChatMessageBubble JSX element.
 */
export default function ChatMessageBubble({
  message,
  onThumbsUp,
  onThumbsDown,
}: ChatMessageBubbleProps): JSX.Element {
  const isUser = message.sender === "user";
  const [thumbsState, setThumbsState] = useState<"up" | "down" | null>(null);
  const hasFeedbackSelection = thumbsState !== null;

  const displayText = message.isError ? "Something went wrong" : message.text;

  // Safely format timestamp with fallback for invalid dates
  let formattedTime = "--";
  try {
    const dateObj =
      message.timestamp instanceof Date
        ? message.timestamp
        : new Date(message.timestamp);

    if (!Number.isNaN(dateObj.getTime())) {
      formattedTime = dateObj.toLocaleTimeString("en-US", {
        hour: "2-digit",
        minute: "2-digit",
        hour12: true,
      });
    }
  } catch {
    // formattedTime remains "--"
  }

  if (isUser) {
    // User message with avatar
    return (
      <Box
        sx={{
          display: "flex",
          justifyContent: "flex-end",
          gap: 1.5,
          alignItems: "flex-end",
        }}
      >
        <Box sx={{ maxWidth: "80%" }}>
          <Paper
            sx={{
              p: 2,
              bgcolor: "action.hover",
              color: "text.primary",
              borderRadius: (theme) =>
                `${theme.spacing(2)} ${theme.spacing(2)} ${theme.spacing(0.5)} ${theme.spacing(2)}`,
              boxShadow: "none",
              border: "1px solid",
              borderColor: "divider",
            }}
          >
            <Typography variant="body2" sx={{ whiteSpace: "pre-line" }}>
              {displayText}
            </Typography>
          </Paper>
        </Box>
        <Paper
          sx={{
            width: (theme) => theme.spacing(4),
            height: (theme) => theme.spacing(4),
            borderRadius: "50%",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
            bgcolor: "primary.lighter",
            color: "primary.main",
          }}
        >
          <User size={16} />
        </Paper>
      </Box>
    );
  }

  // Bot message - new custom layout
  return (
    <Box sx={{ maxWidth: "80%" }}>
      <Box sx={{ mb: 3 }}>
        {/* Header with icon and Novera label */}
        <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 3 }}>
          <Box
            sx={{
              width: 24,
              height: 24,
              borderRadius: "50%",
              background: "linear-gradient(135deg, #EA580C 0%, #F97316 100%)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexShrink: 0,
            }}
          >
            <Bot size={16} color="white" />
          </Box>
          <Typography
            variant="body2"
            sx={{ fontWeight: 600, color: "text.primary" }}
          >
            Novera
          </Typography>
        </Box>

        {/* Message content */}
        {message.isError ? (
          <Typography variant="body2" color="error.main">
            {displayText}
          </Typography>
        ) : (
          <Box
            sx={{
              "& h1:first-of-type, & h2:first-of-type, & h3:first-of-type, & p:first-of-type":
                { mt: 0 },
            }}
            className="prose prose-sm max-w-none text-gray-800"
          >
            <ReactMarkdown components={markdownComponents}>
              {displayText}
            </ReactMarkdown>
          </Box>
        )}

        {/* Thumbs up/down and time - only if no recommendations */}
        {(!message.recommendations || message.recommendations.length === 0) && (
          <Box sx={{ display: "flex", alignItems: "center", gap: 3, mt: 3 }}>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <IconButton
                size="small"
                onClick={() => {
                  if (hasFeedbackSelection) return;
                  setThumbsState("up");
                  onThumbsUp?.(message.id);
                }}
                aria-pressed={thumbsState === "up"}
                sx={{
                  p: 0.5,
                  color:
                    thumbsState === "up" ? "success.main" : "text.secondary",
                  "&:hover": {
                    color: "success.main",
                    bgcolor: "success.lighter",
                  },
                }}
                aria-label="Like this response"
              >
                <ThumbsUp size={14} />
              </IconButton>
              <IconButton
                size="small"
                onClick={() => {
                  if (hasFeedbackSelection) return;
                  setThumbsState("down");
                  onThumbsDown?.(message.id);
                }}
                aria-pressed={thumbsState === "down"}
                sx={{
                  p: 0.5,
                  color:
                    thumbsState === "down" ? "error.main" : "text.secondary",
                  "&:hover": {
                    color: "error.main",
                    bgcolor: "error.lighter",
                  },
                }}
                aria-label="Dislike this response"
              >
                <ThumbsDown size={14} />
              </IconButton>
            </Box>
            <Typography variant="caption" color="text.secondary">
              {formattedTime}
            </Typography>
          </Box>
        )}
      </Box>

      {/* Recommendations - shown after message content */}
      {!isUser &&
        message.recommendations &&
        message.recommendations.length > 0 && (
          <RecommendationsCard recommendations={message.recommendations} />
        )}
    </Box>
  );
}
