import { useEffect, useState } from "react";
import { fetchSummary, type Summary } from "../services/api";

export function useSummary() {
  const [data, setData] = useState<Summary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const summary = await fetchSummary();
      setData(summary);
    } catch (e) {
      setError(e instanceof Error ? e.message : "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return { data, error, loading, reload: load };
}
