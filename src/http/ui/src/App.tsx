import React from "react";
import {
  AppBar,
  Box,
  CssBaseline,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  ThemeProvider,
  Divider,
} from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";
import SettingsIcon from "@mui/icons-material/Settings";
import DashboardIcon from "@mui/icons-material/Dashboard";
import LanguageIcon from "@mui/icons-material/Language";
import Logs from "./components/Logs";
import Domains from "./components/Domains";
import Settings from "./components/Settings";
import { theme, colors } from "./Theme";

const DRAWER_WIDTH = 240;

export default function App() {
  const [drawerOpen, setDrawerOpen] = React.useState(true);
  const [currentView, setCurrentView] = React.useState<
    "logs" | "domains" | "settings"
  >("domains");

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: "flex", height: "100vh" }}>
        {/* Left Drawer */}
        <Drawer
          variant="persistent"
          open={drawerOpen}
          sx={{
            width: DRAWER_WIDTH,
            flexShrink: 0,
            "& .MuiDrawer-paper": {
              width: DRAWER_WIDTH,
              boxSizing: "border-box",
            },
          }}
        >
          <Toolbar>
            <Typography variant="h6">
              <Typography
                component="span"
                sx={{ fontWeight: 800, color: colors.secondary }}
              >
                B4
              </Typography>
              :
              <Typography
                component="span"
                sx={{ fontWeight: 800, color: colors.primary }}
              >
                B
              </Typography>
              ye
              <Typography
                component="span"
                sx={{ fontWeight: 800, color: colors.primary }}
              >
                B
              </Typography>
              ye
              <Typography
                component="span"
                sx={{ fontWeight: 800, color: colors.primary }}
              >
                B
              </Typography>
              ig
              <Typography
                component="span"
                sx={{ fontWeight: 800, color: colors.primary }}
              >
                B
              </Typography>
              ro
            </Typography>
          </Toolbar>
          <Divider sx={{ borderColor: colors.border.default }} />
          <List>
            <ListItem disablePadding>
              <ListItemButton
                selected={currentView === "domains"}
                onClick={() => setCurrentView("domains")}
                sx={{
                  "&.Mui-selected": {
                    backgroundColor: colors.accent.primary,
                    "&:hover": {
                      backgroundColor: colors.accent.primaryHover,
                    },
                  },
                }}
              >
                <ListItemIcon sx={{ color: "inherit" }}>
                  <LanguageIcon />
                </ListItemIcon>
                <ListItemText primary="Domains" />
              </ListItemButton>
            </ListItem>
            <ListItem disablePadding>
              <ListItemButton
                selected={currentView === "logs"}
                onClick={() => setCurrentView("logs")}
                sx={{
                  "&.Mui-selected": {
                    backgroundColor: colors.accent.primary,
                    "&:hover": {
                      backgroundColor: colors.accent.primaryHover,
                    },
                  },
                }}
              >
                <ListItemIcon sx={{ color: "inherit" }}>
                  <DashboardIcon />
                </ListItemIcon>
                <ListItemText primary="Logs" />
              </ListItemButton>
            </ListItem>
            <ListItem disablePadding>
              <ListItemButton
                selected={currentView === "settings"}
                onClick={() => setCurrentView("settings")}
                sx={{
                  "&.Mui-selected": {
                    backgroundColor: colors.accent.primary,
                    "&:hover": {
                      backgroundColor: colors.accent.primaryHover,
                    },
                  },
                }}
              >
                <ListItemIcon sx={{ color: "inherit" }}>
                  <SettingsIcon />
                </ListItemIcon>
                <ListItemText primary="Settings" />
              </ListItemButton>
            </ListItem>
          </List>
        </Drawer>

        {/* Main Content */}
        <Box
          component="main"
          sx={{
            flexGrow: 1,
            display: "flex",
            flexDirection: "column",
            height: "100vh",
            ml: drawerOpen ? 0 : `-${DRAWER_WIDTH}px`,
            transition: theme.transitions.create("margin", {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.leavingScreen,
            }),
          }}
        >
          {/* AppBar */}
          <AppBar position="static" elevation={0}>
            <Toolbar>
              <IconButton
                color="inherit"
                onClick={() => setDrawerOpen(!drawerOpen)}
                edge="start"
                sx={{ mr: 2 }}
              >
                <MenuIcon />
              </IconButton>
              <Typography variant="h6" sx={{ flexGrow: 1, fontWeight: 600 }}>
                {currentView === "logs"
                  ? "Log Viewer"
                  : currentView === "domains"
                  ? "Domain Connections"
                  : "Settings"}
              </Typography>
            </Toolbar>
          </AppBar>

          {/* Content Area */}
          {currentView === "logs" ? (
            <Logs />
          ) : currentView === "domains" ? (
            <Domains />
          ) : (
            <Settings />
          )}
        </Box>
      </Box>
    </ThemeProvider>
  );
}
