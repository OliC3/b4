import { useState } from "react";
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
import ScienceIcon from "@mui/icons-material/Science";
import Dashboard from "@pages/Dashboard";
import Logs from "@pages/Logs";
import Domains from "@pages/Domains";
import Settings from "@pages/Settings";
import Test from "@pages/Checker";
import { theme, colors } from "@design";
import Logo from "@molecules/Logo";
import Version from "@organisms/version/Version";

const DRAWER_WIDTH = 240;

interface NavItem {
  path: string;
  label: string;
  icon: React.ReactNode;
}

const navItems: NavItem[] = [
  { path: "/dashboard", label: "Dashboard", icon: <SpeedIcon /> },
  { path: "/domains", label: "Domains", icon: <LanguageIcon /> },
  { path: "/test", label: "Test", icon: <ScienceIcon /> },
  { path: "/logs", label: "Logs", icon: <AssessmentIcon /> },
  { path: "/settings", label: "Settings", icon: <SettingsIcon /> },
];

export default function App() {
  const [drawerOpen, setDrawerOpen] = useState<boolean>(true);
  const navigate = useNavigate();
  const location = useLocation();

  const getPageTitle = () => {
    const path = location.pathname;
    if (path.startsWith("/dashboard")) return "System Dashboard";
    if (path.startsWith("/domains")) return "Domain Connections";
    if (path.startsWith("/test")) return "DPI Bypass Test";
    if (path.startsWith("/logs")) return "Log Viewer";
    if (path.startsWith("/settings")) return "Settings";
    return "B4";
  };

  const isNavItemSelected = (navPath: string) => {
    if (navPath === "/settings") {
      return location.pathname.startsWith("/settings");
    }
    return location.pathname === navPath;
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: "flex", height: "100vh" }}>
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

          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/domains" element={<Domains />} />
            <Route path="/test" element={<Test />} />
            <Route path="/logs" element={<Logs />} />
            <Route path="/settings/*" element={<Settings />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Box>
      </Box>
    </ThemeProvider>
  );
}
