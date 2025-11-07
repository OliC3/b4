import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
} from "react";

interface WebSocketContextType {
  logs: string[];
  domains: string[];
  pauseLogs: boolean;
  pauseDomains: boolean;
  unseenDomainsCount: number;
  setPauseLogs: (paused: boolean) => void;
  setPauseDomains: (paused: boolean) => void;
  clearLogs: () => void;
  clearDomains: () => void;
  resetDomainsBadge: () => void;
}

const WebSocketContext = createContext<WebSocketContextType | null>(null);

export const WebSocketProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [logs, setLogs] = useState<string[]>([]);
  const [domains, setDomains] = useState<string[]>([]);
  const [pauseLogs, setPauseLogs] = useState(false);
  const [pauseDomains, setPauseDomains] = useState(false);
  const [unseenDomainsCount, setUnseenDomainsCount] = useState(0);

  useEffect(() => {
    const ws = new WebSocket(
      (location.protocol === "https:" ? "wss://" : "ws://") +
        location.host +
        "/api/ws/logs"
    );

    ws.onmessage = (ev) => {
      const line = String(ev.data);
      if (!pauseLogs) {
        setLogs((prev) => [...prev.slice(-999), line]);
      }
    };

    ws.onerror = () => setLogs((prev) => [...prev, "[WS ERROR]"]);

    return () => ws.close();
  }, [pauseLogs]);

  useEffect(() => {
    const ws = new WebSocket(
      (location.protocol === "https:" ? "wss://" : "ws://") +
        location.host +
        "/api/ws/logs"
    );

    ws.onmessage = (ev) => {
      const line = String(ev.data);
      if (!pauseDomains) {
        setDomains((prev) => [...prev.slice(-999), line]);
        setUnseenDomainsCount((count) => count + 1);
      }
    };

    ws.onerror = () => setDomains((prev) => [...prev, "[WS ERROR]"]);

    return () => ws.close();
  }, [pauseDomains]);

  const clearLogs = useCallback(() => setLogs([]), []);
  const clearDomains = useCallback(() => {
    setDomains([]);
    setUnseenDomainsCount(0);
  }, []);

  const resetDomainsBadge = useCallback(() => {
    setUnseenDomainsCount(0);
  }, []);

  return (
    <WebSocketContext.Provider
      value={{
        logs,
        domains,
        pauseLogs,
        pauseDomains,
        unseenDomainsCount,
        setPauseLogs,
        setPauseDomains,
        clearLogs,
        clearDomains,
        resetDomainsBadge,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
};

export const useWebSocket = () => {
  const ctx = useContext(WebSocketContext);
  if (!ctx)
    throw new Error("useWebSocket must be used within WebSocketProvider");
  return ctx;
};
