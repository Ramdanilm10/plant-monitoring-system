import {
  createContext,
  useContext,
  useMemo,
  useState,
} from "react";

import { loginUser } from "../services/api";
import {
  getStoredAuth,
  removeAuth,
  saveAuth,
} from "../services/authStorage";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [auth, setAuth] = useState(() => {
    return getStoredAuth();
  });

  async function login({
    username,
    password,
    role,
  }) {
    const result = await loginUser({
      username,
      password,
      role,
    });

    const authData = {
      token: result.token,
      user: result.user,
    };

    saveAuth(authData);
    setAuth(authData);

    return authData;
  }

  function logout() {
    removeAuth();
    setAuth(null);
  }

  const contextValue = useMemo(
    () => ({
      auth,
      user: auth?.user || null,
      token: auth?.token || null,
      isAuthenticated: Boolean(auth?.token),
      login,
      logout,
    }),
    [auth],
  );

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error(
      "useAuth harus digunakan di dalam AuthProvider.",
    );
  }

  return context;
}