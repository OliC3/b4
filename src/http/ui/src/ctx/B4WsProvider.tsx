import { parseSniLogLine } from "@/utils/logs";
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
  showAll: boolean;
  pauseDomains: boolean;
  unseenDomainsCount: number;
  setShowAll: (showAll: boolean) => void;
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
  const [showAll, setShowAll] = useState(false);
  const [unseenDomainsCount, setUnseenDomainsCount] = useState(0);
  const [, setBuffer] = useState<string[]>([]);

  useEffect(() => {
    const ws = new WebSocket(
      (location.protocol === "https:" ? "wss://" : "ws://") +
        location.host +
        "/api/ws/logs"
    );

    ws.onmessage = (ev) => {
      const line = String(ev.data);
      setBuffer((prev) => [...prev, line]);
    };
    ws.onerror = () => setLogs((prev) => [...prev, "[WS ERROR]"]);

    const interval = setInterval(() => {
      setBuffer((prev) => {
        if (prev.length === 0) return prev;

        if (!pauseLogs) {
          setLogs((curr) => [...curr.slice(-999), ...prev]);
        }
        if (!pauseDomains) {
          setDomains((curr) => {
            const newDomains = [...curr.slice(-999), ...prev];
            const newCount = prev.reduce((count, line) => {
              const log = parseSniLogLine(line);
              return log?.hostSet || log?.ipSet ? count + 1 : count;
            }, 0);
            setUnseenDomainsCount((c) => c + newCount);
            return newDomains;
          });
        }
        return [];
      });
    }, 100);

    return () => {
      clearInterval(interval);
      ws.close();
    };
  }, [pauseLogs, pauseDomains]);

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
        showAll,
        setShowAll,
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
