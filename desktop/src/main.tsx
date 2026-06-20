// lux desktop — the converged @hanzo/ai app (full native via @hanzo/ai/desktop).
// The brand is passed EXPLICITLY (getChainByName('lux')) rather than guessed
// from VITE_BRAND: this entry consumes a PRE-BUILT lib whose import.meta.env was
// frozen at build time, so a runtime VITE_BRAND would be invisible to getBrand().
import { createRoot } from "react-dom/client";
import HanzoAI, { getChainByName } from "@hanzo/ai/desktop";
import "@hanzo/ai/ai.css";

createRoot(document.getElementById("root")!).render(
  <HanzoAI
    {...getChainByName("lux")}
    features={{ chat: true, wallet: true, mining: true, tools: true, agents: true }}
  />,
);
