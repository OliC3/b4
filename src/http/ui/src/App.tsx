import React from "react";
import {
  Routes,
  Route,
  Navigate,
  useNavigate,
  useLocation,
} from "react-router-dom";
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
import LanguageIcon from "@mui/icons-material/Language";
import SpeedIcon from "@mui/icons-material/Speed";
import AssessmentIcon from "@mui/icons-material/Assessment";
import Dashboard from "./components/pages/Dashboard";
import Logs from "./components/pages/Logs";
import Domains from "./components/pages/Domains";
import Settings from "./components/pages/Settings";
import { theme, colors } from "./Theme";
import Logo from "./components/molecules/Logo";
import Version from "./components/molecules/Version";

const DRAWER_WIDTH = 240;

interface NavItem {
  path: string;
  label: string;
  icon: React.ReactNode;
}

const navItems: NavItem[] = [
  { path: "/dashboard", label: "Dashboard", icon: <SpeedIcon /> },
  { path: "/domains", label: "Domains", icon: <LanguageIcon /> },
  { path: "/logs", label: "Logs", icon: <AssessmentIcon /> },
  { path: "/settings", label: "Settings", icon: <SettingsIcon /> },
];

export default function App() {
  const [drawerOpen, setDrawerOpen] = React.useState(true);
  const navigate = useNavigate();
  const location = useLocation();

  // Get the current page title based on the route
  const getPageTitle = () => {
    const path = location.pathname;
    if (path.startsWith("/dashboard")) return "System Dashboard";
    if (path.startsWith("/domains")) return "Domain Connections";
    if (path.startsWith("/logs")) return "Log Viewer";
    if (path.startsWith("/settings")) return "Settings";
    return "B4";
  };

  // Check if a nav item is selected based on the current path
  const isNavItemSelected = (navPath: string) => {
    // Special handling for settings - it's selected for any settings subpath
    if (navPath === "/settings") {
      return location.pathname.startsWith("/settings");
    }
    return location.pathname === navPath;
  };

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
            <Logo />
          </Toolbar>
          <Divider sx={{ borderColor: colors.border.default }} />
          <List>
            {navItems.map((item) => (
              <ListItem key={item.path} disablePadding>
                <ListItemButton
                  selected={isNavItemSelected(item.path)}
                  onClick={() => navigate(item.path)}
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
                    {item.icon}
                  </ListItemIcon>
                  <ListItemText primary={item.label} />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
          <Box sx={{ flexGrow: 1 }} />
          <Version />
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
                {getPageTitle()}
              </Typography>
            </Toolbar>
          </AppBar>

          {/* Content Area with Routing */}
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/domains" element={<Domains />} />
            <Route path="/logs" element={<Logs />} />
            <Route path="/settings/*" element={<Settings />} />
            {/* Catch all route - redirect to dashboard */}
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Box>
      </Box>
    </ThemeProvider>
  );
}
