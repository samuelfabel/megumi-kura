import { useEffect, useState } from "react";
import { AdminPage } from "./pages/AdminPage";
import { DashboardPage } from "./pages/DashboardPage";

type View = "public" | "admin";

function viewFromHash(): View {
  return window.location.hash.replace(/^#/, "") === "admin" ? "admin" : "public";
}

export default function App() {
  const [view, setView] = useState<View>(() => viewFromHash());

  useEffect(() => {
    const onHash = () => setView(viewFromHash());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  function goAdmin() {
    window.location.hash = "admin";
    setView("admin");
  }

  function goPublic() {
    window.location.hash = "";
    setView("public");
  }

  if (view === "admin") {
    return <AdminPage onBack={goPublic} />;
  }
  return <DashboardPage onAdmin={goAdmin} />;
}
