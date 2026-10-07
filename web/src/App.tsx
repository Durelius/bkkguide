import { lazy, Suspense } from "react";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { AppShell } from "./components/AppShell";
import { CategoryPage } from "./pages/CategoryPage";
import { ComingSoon } from "./pages/ComingSoon";
import { PlacePage } from "./pages/PlacePage";

// MapLibre is ~1 MB, so map pages load it on demand.
const MapHome = lazy(() => import("./pages/MapHome").then((m) => ({ default: m.MapHome })));
const StyleTile = lazy(() => import("./pages/StyleTile").then((m) => ({ default: m.StyleTile })));

export function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={null}>
        <Routes>
          <Route path="/styletile" element={<StyleTile />} />
          <Route element={<AppShell />}>
            <Route index element={<MapHome />} />
            <Route path="c/:slug" element={<CategoryPage />} />
            <Route path="place/:slug" element={<PlacePage />} />
            <Route path="search" element={<ComingSoon title="Search" />} />
            <Route path="favourites" element={<ComingSoon title="Saved places" />} />
            <Route path="*" element={<ComingSoon title="Page not found" />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
