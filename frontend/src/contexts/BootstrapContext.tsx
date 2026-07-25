import { createContext, useContext, useEffect, useState, useMemo, useCallback, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { fetchBootstrap, type BootstrapData } from "@/api/bootstrap";
import { resolveLocale } from "@/utils/locale";

interface BootstrapContextValue {
  data: BootstrapData | null;
  isLoading: boolean;
  locale: string;
  refetch: () => Promise<void>;
}

const BootstrapContext = createContext<BootstrapContextValue>({
  data: null,
  isLoading: true,
  locale: "zh",
  refetch: async () => {},
});

 
export function useBootstrap() {
  return useContext(BootstrapContext);
}

export function BootstrapProvider({ children }: { children: ReactNode }) {
  const { i18n } = useTranslation("common");
  const locale = resolveLocale(i18n.language);
  const [data, setData] = useState<BootstrapData | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const loadBootstrap = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await fetchBootstrap(locale);
      setData(result);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, [locale]);

  useEffect(() => {
    loadBootstrap();
  }, [loadBootstrap]);

  const value = useMemo(() => ({
    data,
    isLoading,
    locale,
    refetch: loadBootstrap,
  }), [data, isLoading, locale, loadBootstrap]);

  return (
    <BootstrapContext.Provider value={value}>
      {children}
    </BootstrapContext.Provider>
  );
}
