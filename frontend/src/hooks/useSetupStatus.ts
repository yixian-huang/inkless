import { useCallback, useEffect, useState } from "react";
import { fetchSetupStatus, type SetupStatus } from "@/api/setup";

let cachedStatus: SetupStatus | null = null;
let inflight: Promise<SetupStatus> | null = null;

async function loadSetupStatus(force = false): Promise<SetupStatus> {
  if (!force && cachedStatus) {
    return cachedStatus;
  }
  if (!inflight) {
    inflight = fetchSetupStatus()
      .then((status) => {
        cachedStatus = status;
        return status;
      })
      .finally(() => {
        inflight = null;
      });
  }
  return inflight;
}

export function clearSetupStatusCache() {
  cachedStatus = null;
  inflight = null;
}

export function useSetupStatus() {
  const [status, setStatus] = useState<SetupStatus | null>(cachedStatus);
  const [loading, setLoading] = useState(!cachedStatus);
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(async (options?: { background?: boolean }) => {
    const background = options?.background === true;
    // Soft/background refresh must NOT flip loading → true, or AdminLayout
    // unmounts <Outlet /> and every admin page loses local editor state.
    if (!background) {
      clearSetupStatusCache();
      setLoading(true);
    }
    try {
      const next = await loadSetupStatus(true);
      setStatus(next);
      setError(null);
      return next;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load setup status");
      throw err;
    } finally {
      if (!background) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    loadSetupStatus()
      .then((next) => {
        if (!cancelled) {
          setStatus(next);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load setup status");
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Re-check install status when the window regains focus, but never block UI.
  useEffect(() => {
    const onFocus = () => {
      void refetch({ background: true }).catch(() => undefined);
    };
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [refetch]);

  return { status, loading, error, refetch };
}
