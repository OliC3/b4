import { createTheme } from "@mui/material";

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
};

export const theme = createTheme({
  palette: {
    mode: "dark",
    primary: { main: colors.primary },
    secondary: { main: colors.secondary },
    info: { main: colors.tertiary },
    error: { main: colors.quaternary },
    background: {
      default: colors.background.default,
      paper: colors.background.paper,
    },
    text: {
      primary: colors.text.primary,
      secondary: colors.text.secondary,
    },
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          // Custom scrollbar styles
          "*::-webkit-scrollbar": {
            width: "12px",
            height: "12px",
          },
          "*::-webkit-scrollbar-track": {
            background: colors.background.default,
            borderRadius: "6px",
          },
          "*::-webkit-scrollbar-thumb": {
            background: `linear-gradient(180deg, ${colors.primary} 0%, ${colors.tertiary} 50%, ${colors.quaternary} 100%)`,
            borderRadius: "6px",
            border: `2px solid ${colors.background.default}`,
            "&:hover": {
              background: `linear-gradient(180deg, ${colors.secondary} 0%, ${colors.primary} 50%, ${colors.tertiary} 100%)`,
            },
          },
          "*::-webkit-scrollbar-thumb:active": {
            background: colors.secondary,
          },
          // Firefox scrollbar
          "*": {
            scrollbarWidth: "thin",
            scrollbarColor: `${colors.primary} ${colors.background.default}`,
          },
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          background: `linear-gradient(90deg, ${colors.quaternary} 0%, ${colors.tertiary} 35%, ${colors.primary} 70%, ${colors.secondary} 100%)`,
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: "none",
          borderColor: colors.border.default,
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: colors.background.paper,
          borderRight: `1px solid ${colors.border.default}`,
        },
      },
    },
    MuiSvgIcon: {
      styleOverrides: {
        root: {
          variants: [
            {
              props: { color: "tertiary" },
              style: {
                color: colors.tertiary,
              },
            },
            {
              props: { color: "quaternary" },
              style: {
                color: colors.quaternary,
              },
            },
          ],
        },
      },
    },
  },
  typography: {
    fontFamily:
      'system-ui, -apple-system, "Segoe UI", Roboto, Ubuntu, "Helvetica Neue", Arial',
  },
});
