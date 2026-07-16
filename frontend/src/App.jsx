import {
  Navigate,
  Route,
  Routes,
} from "react-router";

import ProtectedRoute from "./components/ProtectedRoute";
import AppLayout from "./layouts/AppLayout";

import AdminPlants from "./pages/AdminPlants";
import Dashboard from "./pages/Dashboard";
import Login from "./pages/Login";
import RoleSelection from "./pages/RoleSelection";

function App() {
  return (
    <Routes>
      <Route
        path="/"
        element={<RoleSelection />}
      />

      <Route
        path="/login/:role"
        element={<Login />}
      />

      <Route
        element={
          <ProtectedRoute
            allowedRoles={[
              "admin",
              "viewer",
            ]}
          />
        }
      >
        <Route element={<AppLayout />}>
          <Route
            path="/dashboard"
            element={<Dashboard />}
          />

          <Route
            element={
              <ProtectedRoute
                allowedRoles={["admin"]}
              />
            }
          >
            <Route
              path="/admin/plants"
              element={<AdminPlants />}
            />
          </Route>
        </Route>
      </Route>

      <Route
        path="*"
        element={
          <Navigate
            to="/"
            replace
          />
        }
      />
    </Routes>
  );
}

export default App;