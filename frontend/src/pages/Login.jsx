import {
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  Navigate,
  useNavigate,
  useParams,
} from "react-router";

import { useAuth } from "../contexts/AuthContext";

const allowedRoles = ["admin", "viewer"];

function Login() {
  const { role } = useParams();
  const navigate = useNavigate();

  const {
    isAuthenticated,
    login,
  } = useAuth();

  const normalizedRole =
    role?.toLowerCase() || "";

  const isValidRole =
    allowedRoles.includes(normalizedRole);

  const roleLabel = useMemo(() => {
    return normalizedRole === "admin"
      ? "Admin"
      : "Viewer";
  }, [normalizedRole]);

  const [formData, setFormData] = useState({
    username: "",
    password: "",
  });

  const [errorMessage, setErrorMessage] =
    useState("");

  const [isSubmitting, setIsSubmitting] =
    useState(false);

  useEffect(() => {
    setErrorMessage("");
    setFormData({
      username: "",
      password: "",
    });
  }, [normalizedRole]);

  if (!isValidRole) {
    return <Navigate to="/" replace />;
  }

  if (isAuthenticated) {
    return (
      <Navigate
        to="/dashboard"
        replace
      />
    );
  }

  function handleInputChange(event) {
    const { name, value } = event.target;

    setFormData((currentData) => ({
      ...currentData,
      [name]: value,
    }));
  }

  async function handleSubmit(event) {
    event.preventDefault();

    setErrorMessage("");

    const username =
      formData.username.trim();

    const password = formData.password;

    if (!username || !password) {
      setErrorMessage(
        "Username dan password wajib diisi.",
      );
      return;
    }

    setIsSubmitting(true);

    try {
      await login({
        username,
        password,
        role: normalizedRole,
      });

      navigate("/dashboard", {
        replace: true,
      });
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Login gagal diproses.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen bg-slate-100">
      <div className="mx-auto flex min-h-screen max-w-lg items-center px-5 py-12">
        <section className="w-full rounded-3xl border border-slate-200 bg-white p-7 shadow-sm sm:p-9">
          <button
            type="button"
            onClick={() => navigate("/")}
            className="text-sm font-semibold text-slate-500 transition hover:text-slate-900"
          >
            ← Kembali pilih akses
          </button>

          <div className="mt-8">
            <p className="text-sm font-semibold uppercase tracking-[0.2em] text-emerald-700">
              Login {roleLabel}
            </p>

            <h1 className="mt-3 text-3xl font-semibold tracking-tight text-slate-950">
              Masuk ke sistem
            </h1>

            <p className="mt-3 text-sm leading-6 text-slate-600">
              Masukkan akun yang terdaftar sebagai{" "}
              {roleLabel}.
            </p>
          </div>

          <form
            onSubmit={handleSubmit}
            className="mt-8 space-y-5"
          >
            <div>
              <label
                htmlFor="username"
                className="text-sm font-semibold text-slate-700"
              >
                Username
              </label>

              <input
                id="username"
                name="username"
                type="text"
                autoComplete="username"
                value={formData.username}
                onChange={handleInputChange}
                placeholder="Masukkan username"
                disabled={isSubmitting}
                className="mt-2 h-12 w-full rounded-xl border border-slate-300 bg-white px-4 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100 disabled:cursor-not-allowed disabled:bg-slate-100"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="text-sm font-semibold text-slate-700"
              >
                Password
              </label>

              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                value={formData.password}
                onChange={handleInputChange}
                placeholder="Masukkan password"
                disabled={isSubmitting}
                className="mt-2 h-12 w-full rounded-xl border border-slate-300 bg-white px-4 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100 disabled:cursor-not-allowed disabled:bg-slate-100"
              />
            </div>

            {errorMessage && (
              <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3">
                <p className="text-sm leading-6 text-red-700">
                  {errorMessage}
                </p>
              </div>
            )}

            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-12 w-full items-center justify-center rounded-xl bg-slate-900 px-5 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isSubmitting
                ? "Memproses login..."
                : `Masuk sebagai ${roleLabel}`}
            </button>
          </form>
        </section>
      </div>
    </main>
  );
}

export default Login;