import { useEffect, useState } from "react";
import type { PublicView } from "./components/PublicNav";
import { AdminPage } from "./pages/AdminPage";
import { HomePage } from "./pages/HomePage";
import { PromisePage } from "./pages/PromisePage";
import { StockPage } from "./pages/StockPage";

type View = PublicView | "admin";

function viewFromHash(): View {
  const h = window.location.hash.replace(/^#/, "").split("/")[0];
  if (h === "admin" || h.startsWith("admin")) return "admin";
  if (h === "estoque" || h === "stock") return "stock";
  if (h === "doar" || h === "promessas" || h === "promise") return "promise";
  return "home";
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

  function goPublic(next: PublicView = "home") {
    const hash = next === "home" ? "" : next === "stock" ? "estoque" : "doar";
    window.location.hash = hash;
    setView(next);
  }

  if (view === "admin") {
    return <AdminPage onBack={() => goPublic("home")} />;
  }
  if (view === "stock") {
    return <StockPage onNavigate={goPublic} onAdmin={goAdmin} />;
  }
  if (view === "promise") {
    return <PromisePage onNavigate={goPublic} onAdmin={goAdmin} />;
  }
  return <HomePage onNavigate={goPublic} onAdmin={goAdmin} />;
}
