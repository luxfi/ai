import { useState, useEffect, useRef } from "react";
import { invoke } from "@tauri-apps/api/core";
import { Cpu, MessageSquare, Settings, Coins, Send, Bot, User, Loader2 } from "lucide-react";

interface Message {
  role: "user" | "assistant";
  content: string;
}

interface MinerStatus {
  running: boolean;
  tasks_completed: number;
  total_rewards: number;
  gpu_utilization: number;
}

type Tab = "chat" | "miner" | "settings";

function App() {
  const [tab, setTab] = useState<Tab>("chat");
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [model, setModel] = useState("zen-mini-0.5b");
  const [models, setModels] = useState<string[]>([]);
  const [minerStatus, setMinerStatus] = useState<MinerStatus | null>(null);
  const [wallet, setWallet] = useState("");
  const [nodeUrl, setNodeUrl] = useState("http://localhost:9090");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadModels();
    updateMinerStatus();
    const interval = setInterval(updateMinerStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  async function loadModels() {
    try {
      const modelList = await invoke<string[]>("get_models");
      setModels(modelList);
      if (modelList.length > 0 && !modelList.includes(model)) {
        setModel(modelList[0]);
      }
    } catch (e) {
      console.error("Failed to load models:", e);
    }
  }

  async function updateMinerStatus() {
    try {
      const status = await invoke<MinerStatus>("get_miner_status");
      setMinerStatus(status);
    } catch (e) {
      console.error("Failed to get miner status:", e);
    }
  }

  async function sendMessage() {
    if (!input.trim() || loading) return;

    const userMessage: Message = { role: "user", content: input };
    setMessages((prev) => [...prev, userMessage]);
    setInput("");
    setLoading(true);

    try {
      const response = await invoke<{
        choices: Array<{ message: { content: string } }>;
      }>("chat", {
        request: {
          model,
          messages: [...messages, userMessage].map((m) => ({
            role: m.role,
            content: m.content,
          })),
          max_tokens: 2048,
          temperature: 0.7,
        },
      });

      if (response.choices?.[0]?.message?.content) {
        setMessages((prev) => [
          ...prev,
          { role: "assistant", content: response.choices[0].message.content },
        ]);
      }
    } catch (e) {
      console.error("Chat error:", e);
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: `Error: ${e}` },
      ]);
    } finally {
      setLoading(false);
    }
  }

  async function startMiner() {
    if (!wallet) {
      alert("Please enter your wallet address");
      return;
    }
    try {
      await invoke("set_node_url", { url: nodeUrl });
      await invoke<string>("start_miner", { wallet });
      updateMinerStatus();
    } catch (e) {
      console.error("Failed to start miner:", e);
    }
  }

  async function stopMiner() {
    try {
      await invoke<string>("stop_miner");
      updateMinerStatus();
    } catch (e) {
      console.error("Failed to stop miner:", e);
    }
  }

  return (
    <div className="flex h-screen bg-gray-900 text-white">
      {/* Sidebar */}
      <div className="w-16 bg-gray-950 flex flex-col items-center py-4 space-y-4">
        <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-blue-500 rounded-lg flex items-center justify-center font-bold text-lg">
          L
        </div>
        <div className="flex-1 flex flex-col space-y-2 mt-4">
          <button
            onClick={() => setTab("chat")}
            className={`p-3 rounded-lg transition-colors ${
              tab === "chat" ? "bg-purple-600" : "hover:bg-gray-800"
            }`}
          >
            <MessageSquare size={20} />
          </button>
          <button
            onClick={() => setTab("miner")}
            className={`p-3 rounded-lg transition-colors ${
              tab === "miner" ? "bg-purple-600" : "hover:bg-gray-800"
            }`}
          >
            <Cpu size={20} />
          </button>
          <button
            onClick={() => setTab("settings")}
            className={`p-3 rounded-lg transition-colors ${
              tab === "settings" ? "bg-purple-600" : "hover:bg-gray-800"
            }`}
          >
            <Settings size={20} />
          </button>
        </div>
        {minerStatus?.running && (
          <div className="w-3 h-3 bg-green-500 rounded-full animate-pulse" title="Mining active" />
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {tab === "chat" && (
          <>
            {/* Chat Header */}
            <div className="h-14 border-b border-gray-800 flex items-center px-4 justify-between">
              <h1 className="text-lg font-semibold">Lux AI Chat</h1>
              <select
                value={model}
                onChange={(e) => setModel(e.target.value)}
                className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm"
              >
                {models.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {messages.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full text-gray-500">
                  <Bot size={48} className="mb-4" />
                  <p>Start a conversation with Lux AI</p>
                </div>
              )}
              {messages.map((msg, i) => (
                <div
                  key={i}
                  className={`flex gap-3 ${msg.role === "user" ? "justify-end" : ""}`}
                >
                  {msg.role === "assistant" && (
                    <div className="w-8 h-8 rounded-full bg-purple-600 flex items-center justify-center flex-shrink-0">
                      <Bot size={16} />
                    </div>
                  )}
                  <div
                    className={`max-w-[70%] rounded-2xl px-4 py-2 ${
                      msg.role === "user"
                        ? "bg-purple-600"
                        : "bg-gray-800"
                    }`}
                  >
                    <p className="whitespace-pre-wrap">{msg.content}</p>
                  </div>
                  {msg.role === "user" && (
                    <div className="w-8 h-8 rounded-full bg-gray-700 flex items-center justify-center flex-shrink-0">
                      <User size={16} />
                    </div>
                  )}
                </div>
              ))}
              {loading && (
                <div className="flex gap-3">
                  <div className="w-8 h-8 rounded-full bg-purple-600 flex items-center justify-center">
                    <Bot size={16} />
                  </div>
                  <div className="bg-gray-800 rounded-2xl px-4 py-2">
                    <Loader2 className="animate-spin" size={20} />
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>

            {/* Input */}
            <div className="p-4 border-t border-gray-800">
              <div className="flex gap-2">
                <input
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && sendMessage()}
                  placeholder="Type a message..."
                  className="flex-1 bg-gray-800 border border-gray-700 rounded-xl px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
                <button
                  onClick={sendMessage}
                  disabled={loading || !input.trim()}
                  className="bg-purple-600 hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed px-4 rounded-xl transition-colors"
                >
                  <Send size={20} />
                </button>
              </div>
            </div>
          </>
        )}

        {tab === "miner" && (
          <div className="flex-1 p-6">
            <h1 className="text-2xl font-bold mb-6">AI Mining</h1>

            {/* Status Cards */}
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="bg-gray-800 rounded-xl p-4">
                <div className="flex items-center gap-2 text-gray-400 mb-2">
                  <Cpu size={16} />
                  <span className="text-sm">Status</span>
                </div>
                <p className="text-xl font-semibold">
                  {minerStatus?.running ? (
                    <span className="text-green-400">Mining</span>
                  ) : (
                    <span className="text-gray-500">Stopped</span>
                  )}
                </p>
              </div>
              <div className="bg-gray-800 rounded-xl p-4">
                <div className="flex items-center gap-2 text-gray-400 mb-2">
                  <Coins size={16} />
                  <span className="text-sm">Rewards</span>
                </div>
                <p className="text-xl font-semibold">
                  {minerStatus?.total_rewards.toFixed(4) ?? "0.0000"} LUX
                </p>
              </div>
              <div className="bg-gray-800 rounded-xl p-4">
                <div className="flex items-center gap-2 text-gray-400 mb-2">
                  <Cpu size={16} />
                  <span className="text-sm">Tasks</span>
                </div>
                <p className="text-xl font-semibold">
                  {minerStatus?.tasks_completed ?? 0}
                </p>
              </div>
            </div>

            {/* Wallet Input */}
            <div className="bg-gray-800 rounded-xl p-6 mb-4">
              <label className="block text-sm text-gray-400 mb-2">Wallet Address</label>
              <input
                type="text"
                value={wallet}
                onChange={(e) => setWallet(e.target.value)}
                placeholder="0x..."
                className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>

            {/* Control Buttons */}
            <div className="flex gap-4">
              {!minerStatus?.running ? (
                <button
                  onClick={startMiner}
                  className="flex-1 bg-purple-600 hover:bg-purple-700 py-3 rounded-xl font-semibold transition-colors"
                >
                  Start Mining
                </button>
              ) : (
                <button
                  onClick={stopMiner}
                  className="flex-1 bg-red-600 hover:bg-red-700 py-3 rounded-xl font-semibold transition-colors"
                >
                  Stop Mining
                </button>
              )}
            </div>

            {/* GPU Info */}
            {minerStatus?.running && minerStatus.gpu_utilization > 0 && (
              <div className="mt-6 bg-gray-800 rounded-xl p-4">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-gray-400">GPU Utilization</span>
                  <span>{(minerStatus.gpu_utilization * 100).toFixed(1)}%</span>
                </div>
                <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-purple-600 transition-all"
                    style={{ width: `${minerStatus.gpu_utilization * 100}%` }}
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {tab === "settings" && (
          <div className="flex-1 p-6">
            <h1 className="text-2xl font-bold mb-6">Settings</h1>

            <div className="space-y-6">
              <div className="bg-gray-800 rounded-xl p-6">
                <h2 className="text-lg font-semibold mb-4">Node Connection</h2>
                <label className="block text-sm text-gray-400 mb-2">AI Node URL</label>
                <input
                  type="text"
                  value={nodeUrl}
                  onChange={(e) => setNodeUrl(e.target.value)}
                  placeholder="http://localhost:9090"
                  className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
                <button
                  onClick={async () => {
                    await invoke("set_node_url", { url: nodeUrl });
                    loadModels();
                  }}
                  className="mt-4 bg-purple-600 hover:bg-purple-700 px-6 py-2 rounded-lg transition-colors"
                >
                  Save & Reconnect
                </button>
              </div>

              <div className="bg-gray-800 rounded-xl p-6">
                <h2 className="text-lg font-semibold mb-4">Available Models</h2>
                {models.length > 0 ? (
                  <ul className="space-y-2">
                    {models.map((m) => (
                      <li key={m} className="flex items-center gap-2 text-gray-300">
                        <div className="w-2 h-2 bg-green-500 rounded-full" />
                        {m}
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-gray-500">No models available. Check node connection.</p>
                )}
              </div>

              <div className="bg-gray-800 rounded-xl p-6">
                <h2 className="text-lg font-semibold mb-2">About</h2>
                <p className="text-gray-400">Lux AI Desktop v0.1.0</p>
                <p className="text-gray-500 text-sm mt-2">
                  Mine AI tokens by contributing GPU compute and chat with AI models
                  running on the Lux network.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
