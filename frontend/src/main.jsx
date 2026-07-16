import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";

import App from "./App.jsx";
import { AuthProvider } from "./contexts/AuthContext.jsx";

import "./index.css";

const rootElement =
  document.getElementById("root");

if (!rootElement) {
  throw new Error(
    'Elemen dengan id "root" tidak ditemukan.',
  );
}

createRoot(rootElement).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <App />
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
);