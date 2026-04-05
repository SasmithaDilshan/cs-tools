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

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
  type JSX,
} from "react";

export interface FloatingNoveraVisibilityContextValue {
  /** When true, the floating Novera chat is hidden (e.g. case details Activity tab). */
  hideForDetailsActivityTab: boolean;
  setHideForDetailsActivityTab: (hide: boolean) => void;
}

const FloatingNoveraVisibilityContext = createContext<
  FloatingNoveraVisibilityContextValue | undefined
>(undefined);

/**
 * Supplies visibility flags for the global floating Novera chat (e.g. hide on Activity tab).
 *
 * @param props - React children.
 * @returns {JSX.Element} Provider element.
 */
export function FloatingNoveraVisibilityProvider({
  children,
}: {
  children: ReactNode;
}): JSX.Element {
  const [hideForDetailsActivityTab, setHideForDetailsActivityTabState] =
    useState(false);

  const setHideForDetailsActivityTab = useCallback((hide: boolean) => {
    setHideForDetailsActivityTabState(hide);
  }, []);

  const value = useMemo(
    () => ({
      hideForDetailsActivityTab,
      setHideForDetailsActivityTab,
    }),
    [hideForDetailsActivityTab, setHideForDetailsActivityTab],
  );

  return (
    <FloatingNoveraVisibilityContext.Provider value={value}>
      {children}
    </FloatingNoveraVisibilityContext.Provider>
  );
}

/**
 * @returns {FloatingNoveraVisibilityContextValue} Floating Novera visibility state and setter.
 */
export function useFloatingNoveraVisibility(): FloatingNoveraVisibilityContextValue {
  const ctx = useContext(FloatingNoveraVisibilityContext);
  if (ctx === undefined) {
    throw new Error(
      "useFloatingNoveraVisibility must be used within FloatingNoveraVisibilityProvider",
    );
  }
  return ctx;
}
