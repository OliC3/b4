import { useState, useCallback, useEffect } from "react";
import B4Config from "@models/Config";

export function useConfigLoad() {
  const [config, setConfig] = useState<B4Config | null>(null);

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const response = await fetch("/api/config");
        if (!response.ok) throw new Error("Failed to load configuration");
        const data = await response.json();
        setConfig(data);
      } catch (error) {
        console.error("Error loading config:", error);
      }
    };

    fetchConfig();
  }, []);

  return { config };
}

interface ResetResponse {
  success: boolean;
  message: string;
}

export const useConfigReset = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const resetConfig = useCallback(async (): Promise<ResetResponse | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch("/api/config/reset", {
        method: "POST",
      });

      const data = await response.json();

      if (!response.ok) {
        const errorMessage = data.message || "Failed to reset configuration";
        setError(errorMessage);
        setLoading(false);
        return data;
      }

      setLoading(false);
      return data;
    } catch (err) {
      console.error("Error resetting config:", err);
      const errorMessage = "Failed to reset configuration";
      setError(errorMessage);
      setLoading(false);
      return null;
    }
  }, []);

  return {
    resetConfig,
    loading,
    error,
  };
};
