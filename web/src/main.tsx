import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import "@fontsource/ibm-plex-sans-thai/thai-400.css";
import "@fontsource/ibm-plex-sans-thai/thai-700.css";
import "./styles/fonts.css";
import "./styles/tokens.css";
import "./styles/base.css";
import { App } from "./App";

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 60_000 } } });

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
