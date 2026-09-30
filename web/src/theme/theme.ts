export type ThemeColors = {
  primary: string;
  secondary: string;
  accent: string;
};

export function applyTheme(colors: ThemeColors) {
  const root = document.documentElement;
  root.style.setProperty("--color-primary", colors.primary);
  root.style.setProperty("--color-secondary", colors.secondary);
  root.style.setProperty("--color-accent", colors.accent);
}
