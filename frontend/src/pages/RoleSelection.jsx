import { Navigate, useNavigate } from "react-router";

import { useAuth } from "../contexts/AuthContext";

const roleOptions = [
  {
    role: "admin",
    title: "Admin",
    description:
      "Mengakses dashboard dan mengelola data sistem.",
    buttonLabel: "Masuk sebagai Admin",
  },
  {
    role: "viewer",
    title: "Viewer",
    description:
      "Melihat dashboard, status tanaman, dan rekomendasi.",
    buttonLabel: "Masuk sebagai Viewer",
  },
];

function RoleSelection() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  if (isAuthenticated) {
    return (
      <Navigate
        to="/dashboard"
        replace
      />
    );
  }

  return (
    <main className="min-h-screen bg-slate-100">
      <div className="mx-auto flex min-h-screen max-w-5xl items-center px-5 py-12 sm:px-8">
        <div className="w-full">
          <header className="mx-auto max-w-2xl text-center">
            <p className="text-sm font-semibold uppercase tracking-[0.2em] text-emerald-700">
              Monitoring Tanaman
            </p>

            <h1 className="mt-4 text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">
              Pilih akses pengguna
            </h1>

            <p className="mt-4 text-sm leading-7 text-slate-600 sm:text-base">
              Pilih jenis akses sebelum masuk ke sistem
              monitoring tanaman.
            </p>
          </header>

          <section className="mx-auto mt-10 grid max-w-3xl gap-5 md:grid-cols-2">
            {roleOptions.map((option) => (
              <article
                key={option.role}
                className="rounded-3xl border border-slate-200 bg-white p-7 shadow-sm"
              >
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-100 text-lg font-bold text-emerald-700">
                  {option.title.charAt(0)}
                </div>

                <h2 className="mt-6 text-2xl font-semibold text-slate-950">
                  {option.title}
                </h2>

                <p className="mt-3 min-h-14 text-sm leading-6 text-slate-600">
                  {option.description}
                </p>

                <button
                  type="button"
                  onClick={() =>
                    navigate(
                      `/login/${option.role}`,
                    )
                  }
                  className="mt-7 inline-flex h-12 w-full items-center justify-center rounded-xl bg-slate-900 px-5 text-sm font-semibold text-white transition hover:bg-slate-800"
                >
                  {option.buttonLabel}
                </button>
              </article>
            ))}
          </section>
        </div>
      </div>
    </main>
  );
}

export default RoleSelection;