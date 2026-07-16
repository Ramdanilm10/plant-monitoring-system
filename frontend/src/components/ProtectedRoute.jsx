import {
  Navigate,
  Outlet,
  useNavigate,
} from "react-router";

import { useAuth } from "../contexts/AuthContext";

function ProtectedRoute({
  allowedRoles = [],
}) {
  const navigate = useNavigate();

  const {
    isAuthenticated,
    user,
    logout,
  } = useAuth();

  if (!isAuthenticated || !user) {
    return <Navigate to="/" replace />;
  }

  if (
    allowedRoles.length > 0 &&
    !allowedRoles.includes(user.role)
  ) {
    return <Navigate to="/" replace />;
  }

  function handleLogout() {
    logout();
    navigate("/", {
      replace: true,
    });
  }

  return (
    <>
      <div className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-5 py-3 sm:px-8">
          <div>
            <p className="text-xs font-medium text-slate-500">
              Login sebagai
            </p>

            <p className="text-sm font-semibold text-slate-900">
              {user.username} · {user.role}
            </p>
          </div>

          <button
            type="button"
            onClick={handleLogout}
            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
          >
            Keluar
          </button>
        </div>
      </div>

      <Outlet />
    </>
  );
}

export default ProtectedRoute;