import { useQuery } from "@tanstack/react-query";
import { Outlet, useLocation } from "react-router-dom";
import { api } from "../api";
import { Footer } from "./Footer";
import { DesktopNav, TabBar, TopBar } from "./Nav";
import "./AppShell.css";

/** Mobile: top bar, content, bottom tabs. Desktop (>= 900px): one top navbar. */
export function AppShell() {
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });
  const cats = categories.data ?? [];
  const { pathname } = useLocation();
  return (
    <div className="shell">
      <div className="shell__top shell__top--mobile">
        <TopBar />
      </div>
      <div className="shell__top shell__top--desktop">
        <DesktopNav categories={cats} />
      </div>
      <main className="shell__main">
        <Outlet />
        {/* The map fills the screen, so the footer lives on scrolling pages only. */}
        {pathname !== "/" && <Footer />}
      </main>
      <div className="shell__tabs">
        <TabBar categories={cats} />
      </div>
    </div>
  );
}
