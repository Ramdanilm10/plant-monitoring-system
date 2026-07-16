import {
  getStoredToken,
  removeAuth,
} from "./authStorage";

const DEFAULT_REQUEST_TIMEOUT_MS =
  15_000;

/**
 * URL backend dapat diisi untuk deployment terpisah.
 *
 * Untuk mode lokal, LAN, dan Cloudflare Tunnel,
 * nilainya dikosongkan sehingga request memakai
 * path relatif:
 *
 * /api/auth/login
 * /api/plants
 * /api/dashboard/1
 */
const API_BASE_URL = String(
  import.meta.env.VITE_API_BASE_URL || "",
)
  .trim()
  .replace(/\/+$/, "");

export function getApiBaseUrl() {
  if (API_BASE_URL) {
    return API_BASE_URL;
  }

  return window.location.origin;
}

async function parseResponse(
  response,
) {
  const responseText =
    await response.text();

  if (!responseText) {
    return null;
  }

  try {
    return JSON.parse(
      responseText,
    );
  } catch {
    return {
      message: responseText,
    };
  }
}

/**
 * Melakukan request menuju API.
 *
 * Saat API_BASE_URL kosong:
 *
 * endpoint:
 * /api/plants
 *
 * request browser:
 * DOMAIN-FRONTEND/api/plants
 *
 * Vite kemudian meneruskan request tersebut
 * menuju backend Go pada localhost:8080.
 */
async function apiRequest(
  endpoint,
  options = {},
) {
  const {
    timeoutMs =
      DEFAULT_REQUEST_TIMEOUT_MS,

    ...fetchOptions
  } = options;

  const token =
    getStoredToken();

  const headers = new Headers(
    fetchOptions.headers || {},
  );

  headers.set(
    "Accept",
    "application/json",
  );

  if (
    fetchOptions.body &&
    !(
      fetchOptions.body instanceof
      FormData
    )
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  if (token) {
    headers.set(
      "Authorization",
      `Bearer ${token}`,
    );
  }

  const controller =
    new AbortController();

  const timeoutId =
    window.setTimeout(
      () => {
        controller.abort();
      },
      timeoutMs,
    );

  let response;

  try {
    response = await fetch(
      `${API_BASE_URL}${endpoint}`,
      {
        ...fetchOptions,

        headers,

        signal:
          controller.signal,
      },
    );
  } catch (error) {
    if (
      error?.name ===
      "AbortError"
    ) {
      throw new Error(
        "Permintaan melewati batas waktu. Pastikan backend dan koneksi internet masih aktif.",
      );
    }

    throw new Error(
      "Tidak dapat terhubung ke server. Pastikan backend, frontend, dan tunnel masih berjalan.",
    );
  } finally {
    window.clearTimeout(
      timeoutId,
    );
  }

  const result =
    await parseResponse(
      response,
    );

  /**
   * Kalau token dashboard sudah kedaluwarsa,
   * hapus session dan kembali ke halaman login.
   */
  if (
    response.status === 401 &&
    token
  ) {
    removeAuth();

    window.location.assign(
      "/",
    );
  }

  if (!response.ok) {
    const error = new Error(
      result?.message ||
        `Permintaan gagal dengan status ${response.status}.`,
    );

    error.status =
      response.status;

    error.data =
      result;

    throw error;
  }

  return result ?? {};
}

export async function loginUser({
  username,
  password,
  role,
}) {
  return apiRequest(
    "/api/auth/login",
    {
      method: "POST",

      body: JSON.stringify({
        username,
        password,
        role,
      }),
    },
  );
}

export async function getDashboard(
  plantId,
) {
  return apiRequest(
    `/api/dashboard/${plantId}`,
    {
      method: "GET",
    },
  );
}

export async function getSensorHistory(
  plantId,
  range = "24h",
) {
  const safeRange =
    encodeURIComponent(
      range,
    );

  return apiRequest(
    `/api/plants/${plantId}/history?range=${safeRange}`,
    {
      method: "GET",
    },
  );
}

export async function getDSSAnalysis(
  plantId,
  range = "24h",
) {
  const safeRange =
    encodeURIComponent(
      range,
    );

  return apiRequest(
    `/api/plants/${plantId}/dss?range=${safeRange}`,
    {
      method: "GET",
    },
  );
}

export async function getAvailablePlants() {
  return apiRequest(
    "/api/plants",
    {
      method: "GET",
    },
  );
}

export async function getPlants() {
  return apiRequest(
    "/api/admin/plants",
    {
      method: "GET",
    },
  );
}

export async function getPlant(
  plantId,
) {
  return apiRequest(
    `/api/admin/plants/${plantId}`,
    {
      method: "GET",
    },
  );
}

export async function createPlant(
  plantData,
) {
  return apiRequest(
    "/api/admin/plants",
    {
      method: "POST",

      body: JSON.stringify(
        plantData,
      ),
    },
  );
}

export async function updatePlant(
  plantId,
  plantData,
) {
  return apiRequest(
    `/api/admin/plants/${plantId}`,
    {
      method: "PUT",

      body: JSON.stringify(
        plantData,
      ),
    },
  );
}

export async function deletePlant(
  plantId,
) {
  return apiRequest(
    `/api/admin/plants/${plantId}`,
    {
      method: "DELETE",
    },
  );
}

export async function checkBackendHealth() {
  return apiRequest(
    "/api/health",
    {
      method: "GET",

      timeoutMs: 5_000,
    },
  );
}

export async function getDeviceStatuses() {
  return apiRequest(
    "/api/admin/devices/status",
    {
      method: "GET",
    },
  );
}