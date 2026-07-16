import {
  NavLink,
  Outlet,
} from "react-router";

import { useAuth } from "../contexts/AuthContext";

function AppLayout() {
  const { user } = useAuth();

  const navigationItems = [
    {
      label: "Dashboard",
      path: "/dashboard",
      roles: ["admin", "viewer"],
    },
    {
      label: "Kelola Tanaman",
      path: "/admin/plants",
      roles: ["admin"],
    },
  ];

  const visibleNavigationItems =
    navigationItems.filter((item) =>
      item.roles.includes(user?.role),
    );

  return (
    <div className="min-h-screen bg-slate-100">
      <div className="mx-auto grid max-w-7xl gap-6 px-5 py-6 sm:px-8 lg:grid-cols-[230px_minmax(0,1fr)]">
        <aside className="h-fit rounded-2xl border border-slate-200 bg-white p-3 shadow-sm">
          <nav className="space-y-1">
            {visibleNavigationItems.map(
              (item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  className={({ isActive }) =>
                    [
                      "block rounded-xl px-4 py-3 text-sm font-semibold transition",
                      isActive
                        ? "bg-slate-900 text-white"
                        : "text-slate-600 hover:bg-slate-100 hover:text-slate-950",
                    ].join(" ")
                  }
                >
                  {item.label}
                </NavLink>
              ),
            )}
          </nav>
        </aside>

        <div className="min-w-0">
          <Outlet />
        </div>
      </div>
    </div>
  );
}

export default AppLayout;