"use client";

import * as React from "react";

// Light-only theme context. next-themes 0.4.6 injects an inline <script>
// that React 19 / Next 16.2 warns about on the client ("Encountered a
// script tag while rendering React component"). The app already forced
// light theme, so the script was a no-op — drop it.

type Theme = "light" | "dark" | "system";

interface ThemeContextValue {
  theme: Theme;
  resolvedTheme: Theme;
  setTheme: (theme: Theme) => void;
}

const ThemeContext = React.createContext<ThemeContextValue>({
  theme: "light",
  resolvedTheme: "light",
  setTheme: () => {},
});

export function ThemeProvider({
  children,
}: {
  children: React.ReactNode;
  attribute?: string;
  defaultTheme?: string;
  forcedTheme?: string;
  enableSystem?: boolean;
  disableTransitionOnChange?: boolean;
}) {
  const value = React.useMemo<ThemeContextValue>(
    () => ({
      theme: "light",
      resolvedTheme: "light",
      setTheme: () => {},
    }),
    []
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  return React.useContext(ThemeContext);
}
