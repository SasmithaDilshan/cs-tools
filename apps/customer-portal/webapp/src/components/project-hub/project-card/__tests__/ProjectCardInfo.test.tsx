// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import ProjectCardInfo from "@components/project-hub/project-card/ProjectCardInfo";

vi.mock("@wso2/oxygen-ui", () => ({
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  Box: ({ children, sx }: any) => (
    <div data-testid="box" style={sx}>
      {children}
    </div>
  ),
  Form: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    CardHeader: ({ title }: any) => (
      <div data-testid="card-header">
        <div data-testid="title">{title}</div>
      </div>
    ),
  },
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  Tooltip: ({ children }: any) => (
    <span data-testid="tooltip">{children}</span>
  ),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  Typography: ({ children, variant, sx }: any) => (
    <span data-testid={`typography-${variant}`} style={sx}>
      {children}
    </span>
  ),
}));

describe("ProjectCardInfo", () => {
  it("renders project title", () => {
    render(<ProjectCardInfo title="My Project" />);
    expect(screen.getByText("My Project")).toBeInTheDocument();
  });

  it("uses h6 typography for title", () => {
    render(<ProjectCardInfo title="Title" />);
    expect(screen.getByTestId("typography-h6")).toBeInTheDocument();
  });

  it("displays '--' fallback for empty title", () => {
    render(<ProjectCardInfo title="" />);
    expect(screen.getByText("--")).toBeInTheDocument();
  });
});
