"use client";

import { useState, useRef, useEffect } from "react";
import { Send, Bot, User, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Message = {
  id: string;
  role: "user" | "assistant";
  content: string;
};

const WELCOME: Message = {
  id: "welcome",
  role: "assistant",
  content:
    "Hola, soy tu asistente médico de Sinapsis. Puedes preguntarme sobre síntomas, medicamentos, recomendaciones de salud o qué esperar en tu próxima consulta. ¿En qué te puedo ayudar?",
};

export default function ChatPage() {
  const [messages, setMessages] = useState<Message[]>([WELCOME]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [conversationID, setConversationID] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  const token = () =>
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  async function sendMessage(e: React.FormEvent) {
    e.preventDefault();
    const question = input.trim();
    if (!question || loading) return;

    const userMsg: Message = {
      id: crypto.randomUUID(),
      role: "user",
      content: question,
    };
    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setLoading(true);
    setError("");

    try {
      const t = token();
      const res = await fetch("http://localhost:8080/api/v1/chat", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(t ? { Authorization: `Bearer ${t}` } : {}),
        },
        body: JSON.stringify({
          question,
          conversation_id: conversationID,
        }),
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setError(data.error ?? "No se pudo obtener respuesta del asistente.");
        return;
      }

      if (data.conversation_id) setConversationID(data.conversation_id);

      const assistantMsg: Message = {
        id: crypto.randomUUID(),
        role: "assistant",
        content: data.answer,
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch {
      setError("Error de conexión con el servidor.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-b border-line bg-shell px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex size-9 items-center justify-center rounded-lg bg-teal/10 text-teal">
            <Bot className="size-5" />
          </div>
          <div>
            <h1 className="font-display text-base font-semibold text-ink">
              Asistente médico
            </h1>
            <p className="text-xs text-muted">
              Información general de salud · No reemplaza una consulta médica
            </p>
          </div>
        </div>
      </div>

      {/* Mensajes */}
      <div className="flex-1 overflow-y-auto px-4 py-6">
        <div className="mx-auto flex max-w-2xl flex-col gap-4">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={cn(
                "flex gap-3",
                msg.role === "user" ? "flex-row-reverse" : "flex-row",
              )}
            >
              {/* Avatar */}
              <div
                className={cn(
                  "flex size-8 shrink-0 items-center justify-center rounded-full",
                  msg.role === "user"
                    ? "bg-navy text-white"
                    : "bg-teal/10 text-teal",
                )}
              >
                {msg.role === "user" ? (
                  <User className="size-4" />
                ) : (
                  <Bot className="size-4" />
                )}
              </div>

              {/* Burbuja */}
              <div
                className={cn(
                  "max-w-[78%] rounded-2xl px-4 py-3 text-sm leading-relaxed",
                  msg.role === "user"
                    ? "rounded-tr-sm bg-navy text-white"
                    : "rounded-tl-sm border border-line bg-surface text-slate",
                )}
              >
                {msg.content}
              </div>
            </div>
          ))}

          {/* Indicador de escritura */}
          {loading && (
            <div className="flex gap-3">
              <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-teal/10 text-teal">
                <Bot className="size-4" />
              </div>
              <div className="flex items-center gap-1.5 rounded-2xl rounded-tl-sm border border-line bg-surface px-4 py-3">
                <span className="size-1.5 animate-bounce rounded-full bg-muted [animation-delay:0ms]" />
                <span className="size-1.5 animate-bounce rounded-full bg-muted [animation-delay:150ms]" />
                <span className="size-1.5 animate-bounce rounded-full bg-muted [animation-delay:300ms]" />
              </div>
            </div>
          )}

          {/* Error inline */}
          {error && (
            <div className="flex items-center gap-2 rounded-xl border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger">
              <AlertCircle className="size-4 shrink-0" />
              {error}
            </div>
          )}

          <div ref={bottomRef} />
        </div>
      </div>

      {/* Input */}
      <div className="border-t border-line bg-shell px-4 py-4">
        <form
          onSubmit={sendMessage}
          className="mx-auto flex max-w-2xl items-end gap-3"
        >
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendMessage(e as unknown as React.FormEvent);
              }
            }}
            placeholder="Escribe tu pregunta… (Enter para enviar)"
            rows={1}
            className="flex-1 resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20 placeholder:text-muted"
            style={{ maxHeight: "8rem", overflowY: "auto" }}
          />
          <Button
            type="submit"
            disabled={loading || !input.trim()}
            size="md"
            className="shrink-0"
          >
            <Send className="size-4" />
            Enviar
          </Button>
        </form>
      </div>
    </div>
  );
}
