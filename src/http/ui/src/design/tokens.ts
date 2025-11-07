// Color palette
export const colors = {
  primary: "#9E1C60",
  secondary: "#F5AD18",
  tertiary: "#811844",
  quaternary: "#561530",
  background: {
    default: "#1a0e15",
    paper: "#1f1218",
    dark: "#0f0a0e",
    control: "rgba(31, 18, 24, 0.6)",
  },
  text: {
    primary: "#ffe8f4",
    secondary: "#f8d7e9",
  },
  border: {
    default: "rgba(245, 173, 24, 0.24)",
    light: "rgba(245, 173, 24, 0.12)",
    medium: "rgba(245, 173, 24, 0.24)",
    strong: "rgba(245, 173, 24, 0.5)",
  },
  accent: {
    primary: "rgba(158, 28, 96, 0.2)",
    primaryHover: "rgba(158, 28, 96, 0.3)",
    primaryStrong: "rgba(158, 28, 96, 0.1)",
    secondary: "rgba(245, 173, 24, 0.2)",
    secondaryHover: "rgba(245, 173, 24, 0.1)",
    tertiary: "rgba(129, 24, 68, 0.2)",
  },
} as const;

export const spacing = {
  xs: 0.5,
  sm: 1,
  md: 2,
  lg: 3,
  xl: 4,
  xxl: 6,
} as const;

export const radius = {
  sm: 1,
  md: 2,
  lg: 3,
  xl: 4,
} as const;

export const typography = {
  sizes: {
    xs: "0.65rem",
    sm: "0.75rem",
    md: "0.875rem",
    lg: "1rem",
    xl: "1.25rem",
  },
  weights: {
    regular: 400,
    medium: 500,
    semibold: 600,
    bold: 700,
  },
} as const;

export const button_primary = {
  bgcolor: colors.primary,
  "&:hover": { bgcolor: colors.secondary },
} as const;

export const button_secondary = {
  color: colors.text.secondary,
  "&:hover": {
    bgcolor: colors.accent.primaryHover,
  },
} as const;
