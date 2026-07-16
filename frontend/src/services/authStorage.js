const AUTH_STORAGE_KEY = "plant_monitoring_auth";

export function getStoredAuth() {
  const storedAuth = sessionStorage.getItem(
    AUTH_STORAGE_KEY,
  );

  if (!storedAuth) {
    return null;
  }

  try {
    const parsedAuth = JSON.parse(storedAuth);

    if (
      !parsedAuth?.token ||
      !parsedAuth?.user?.username ||
      !parsedAuth?.user?.role
    ) {
      sessionStorage.removeItem(
        AUTH_STORAGE_KEY,
      );

      return null;
    }

    return parsedAuth;
  } catch {
    sessionStorage.removeItem(
      AUTH_STORAGE_KEY,
    );

    return null;
  }
}

export function saveAuth(authData) {
  sessionStorage.setItem(
    AUTH_STORAGE_KEY,
    JSON.stringify(authData),
  );
}

export function removeAuth() {
  sessionStorage.removeItem(
    AUTH_STORAGE_KEY,
  );
}

export function getStoredToken() {
  const authData = getStoredAuth();

  return authData?.token || null;
}