"use client";

import * as React from "react";
import { ThemeProvider as NextThemesProvider, useTheme as useNextTheme } from "next-themes";

type ThemeProviderProps = React.ComponentProps<typeof NextThemesProvider>;

export function ThemeProvider({ children, ...props }: ThemeProviderProps) {
  return <NextThemesProvider {...props}>{children}</NextThemesProvider>;
}

export const useTheme = () => {
  const { theme, setTheme, systemTheme } = useNextTheme();
  
  const currentTheme = theme === "system" ? systemTheme : theme;
  
  const toggleTheme = () => {
    setTheme(currentTheme === "light" ? "dark" : "light");
  };

  return {
    theme: currentTheme || "light",
    setTheme,
    toggleTheme,
  };
};
