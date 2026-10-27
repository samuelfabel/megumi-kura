import { useEffect, useState } from "react";
import { AdminPage } from "./pages/AdminPage";
import { DashboardPage } from "./pages/DashboardPage";
import { SocialPage } from "./pages/SocialPage";

type View = "public" | "admin" | "social";

function viewFromHash(): View {
  const h = window.location.hash.replace(/^#/, "");
  if (h === "admin" || h.startsWith("admin")) return "admin";
  if (h === "social" || h.startsWith("social")) return "social";
  return "public";
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

  function goSocial() {
    window.location.hash = "social";
    setView("social");
  }

  if (view === "admin") {
    return <AdminPage onBack={goPublic} />;
  }
  if (view === "social") {
    return <SocialPage onBack={goPublic} />;
  }
  return <DashboardPage onAdmin={goAdmin} onSocial={goSocial} />;
}
