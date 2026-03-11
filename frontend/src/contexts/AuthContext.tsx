import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from "react";
import axios from "axios";
import { http } from "@/api/http";

interface User {
  id: string;
  username: string;
  role: "admin" | "editor";
  isSuperAdmin: boolean;
  permissions: string[];
}

interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
  hasPermission: (perm: string) => boolean;
}

interface LoginResponse {
  accessToken: string;
  refreshToken: string;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const clearTokens = useCallback(() => {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    setUser(null);
  }, []);

  const refreshTokenInternal = useCallback(async () => {
    const refreshTokenValue = localStorage.getItem("refreshToken");
    if (!refreshTokenValue) {
      throw new Error("No refresh token available");
    }

    try {
      const response = await http.post<LoginResponse>("/auth/refresh", {
        refreshToken: refreshTokenValue,
      });
      localStorage.setItem("accessToken", response.data.accessToken);
    } catch {
      clearTokens();
      throw new Error("Token refresh failed");
    }
  }, [clearTokens]);

  // Session restore on mount
  useEffect(() => {
    const restoreSession = async () => {
      const accessToken = localStorage.getItem("accessToken");
      if (!accessToken) {
        setIsLoading(false);
        return;
      }

      try {
        const response = await http.get<User>("/auth/me", {
          headers: {
            "Authorization": `Bearer ${accessToken}`,
          },
        });
        setUser(response.data);
      } catch (error) {
        const refreshTokenValue = localStorage.getItem("refreshToken");
        if (refreshTokenValue) {
          try {
            await refreshTokenInternal();
            const retryResponse = await http.get<User>("/auth/me", {
              headers: {
                "Authorization": `Bearer ${localStorage.getItem("accessToken")}`,
              },
            });
            setUser(retryResponse.data);
          } catch (retryError) {
            console.error("Session restore failed:", retryError);
            clearTokens();
          }
        } else {
          console.error("Session restore failed:", error);
          clearTokens();
        }
      } finally {
        setIsLoading(false);
      }
    };

    restoreSession();
  }, [clearTokens, refreshTokenInternal]);

  const login = async (username: string, password: string) => {
    try {
      const response = await http.post<LoginResponse>("/auth/login", {
        username,
        password,
      });

      // Store tokens
      localStorage.setItem("accessToken", response.data.accessToken);
      localStorage.setItem("refreshToken", response.data.refreshToken);

      // Fetch user info
      const meResponse = await http.get<User>("/auth/me", {
        headers: {
          "Authorization": `Bearer ${response.data.accessToken}`,
        },
      });
      setUser(meResponse.data);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const errorData = error.response?.data as { error?: { message?: string } } | undefined;
        throw new Error(errorData?.error?.message || "登录失败");
      }
      throw error;
    }
  };

  const refreshToken = async () => {
    await refreshTokenInternal();
  };

  const logout = async () => {
    const refreshTokenValue = localStorage.getItem("refreshToken");
    if (refreshTokenValue) {
      try {
        await http.post("/auth/logout", {
          refreshToken: refreshTokenValue,
        });
      } catch (error) {
        console.error("Logout API call failed:", error);
      }
    }
    clearTokens();
  };

  const hasPermission = useCallback(
    (perm: string): boolean => {
      if (!user) return false;
      if (user.isSuperAdmin) return true;
      return user.permissions?.includes(perm) ?? false;
    },
    [user]
  );

  const value: AuthContextValue = {
    user,
    isAuthenticated: !!user,
    isLoading,
    login,
    logout,
    refreshToken,
    hasPermission,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
