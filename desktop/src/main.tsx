// lux desktop — the converged @hanzo/ai app (full native via @hanzo/ai/desktop).
import { createRoot } from "react-dom/client";
import HanzoAI, { getBrand } from "@hanzo/ai/desktop";
import "@hanzo/ai/ai.css";

createRoot(document.getElementById("root")!).render(
  <HanzoAI {...getBrand()} features={{ chat: true, wallet: true, mining: true, tools: true, agents: true }} />,
);
