"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { TrustBadge } from "./TrustBadge";
import { VerificationPanel } from "./VerificationPanel";
import type { Message } from "@/lib/store";
import { User, Bot, Copy, Check, ChevronRight, Brain, Gauge, Clock, Hash } from "lucide-react";
import { useState, useCallback } from "react";

function CodeBlock({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const language = className?.replace("language-", "") || "";
  const code = String(children).replace(/\n$/, "");

  const copyCode = useCallback(() => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [code]);

  return (
    <div className="relative group my-3">
      <div className="flex items-center justify-between px-3 py-1.5 bg-bg-tertiary rounded-t-lg border border-b-0 border-border-dim">
        <span className="text-[10px] font-mono text-text-tertiary uppercase tracking-wider">
          {language || "code"}
        </span>
        <button
          onClick={copyCode}
          className="flex items-center gap-1 text-[10px] font-mono text-text-tertiary hover:text-text-secondary transition-colors"
        >
          {copied ? <Check size={10} /> : <Copy size={10} />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="!mt-0 !rounded-t-none">
        <code className={className}>{children}</code>
      </pre>
    </div>
  );
}

function ThinkingBlock({
  thinking,
  streaming,
}: {
  thinking: string;
  streaming?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="mb-3">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-accent-amber/8 border border-accent-amber/15 hover:bg-accent-amber/12 transition-all text-accent-amber group"
      >
        <ChevronRight
          size={12}
          className={`transition-transform duration-200 ${
            expanded ? "rotate-90" : ""
          }`}
        />
        <Brain size={12} />
        <span className="text-[11px] font-mono">
          {streaming && !thinking.length
            ? "Thinking..."
            : `Thinking${streaming ? "..." : ""}`}
        </span>
        {!expanded && thinking.length > 0 && (
          <span className="text-[10px] text-text-tertiary ml-1">
            ({thinking.length} chars)
          </span>
        )}
      </button>

      {expanded && (
        <div className="mt-1.5 ml-1 pl-3 border-l-2 border-accent-amber/20">
          <div
            className={`prose text-text-secondary text-[13px] leading-relaxed opacity-80 ${
              streaming ? "streaming-cursor" : ""
            }`}
          >
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {thinking}
            </ReactMarkdown>
          </div>
        </div>
      )}
    </div>
  );
}

function StreamMetrics({
  tps,
  ttft,
  tokenCount,
  streaming,
}: {
  tps?: number;
  ttft?: number;
  tokenCount?: number;
  streaming?: boolean;
}) {
  if (!tps && !ttft) return null;

  return (
    <div
      className={`flex items-center gap-3 mt-2 py-1.5 px-2.5 rounded-lg text-[11px] font-mono ${
        streaming
          ? "bg-accent-green/8 border border-accent-green/15"
          : "bg-bg-tertiary border border-border-dim"
      }`}
    >
      <span
        className={`flex items-center gap-1 ${
          streaming ? "text-accent-green" : "text-text-secondary"
        }`}
      >
        <Gauge size={11} />
        <span className="tabular-nums font-semibold">
          {tps ? tps.toFixed(1) : "—"}
        </span>
        <span className="text-text-tertiary">tok/s</span>
      </span>

      <span className="text-border-default">|</span>

      <span
        className={`flex items-center gap-1 ${
          streaming ? "text-accent-amber" : "text-text-secondary"
        }`}
      >
        <Clock size={11} />
        <span className="tabular-nums font-semibold">
          {ttft ? (ttft < 1000 ? `${Math.round(ttft)}ms` : `${(ttft / 1000).toFixed(2)}s`) : "—"}
        </span>
        <span className="text-text-tertiary">TTFT</span>
      </span>

      <span className="text-border-default">|</span>

      <span className="flex items-center gap-1 text-text-secondary">
        <Hash size={11} />
        <span className="tabular-nums font-semibold">
          {tokenCount || 0}
        </span>
        <span className="text-text-tertiary">tokens</span>
      </span>

      {streaming && (
        <span className="ml-auto flex items-center gap-1 text-accent-green">
          <span className="w-1.5 h-1.5 rounded-full bg-accent-green animate-pulse" />
          <span className="text-[10px]">live</span>
        </span>
      )}
    </div>
  );
}

/* eslint-disable @typescript-eslint/no-explicit-any */
const markdownComponents: any = {
  code({ className, children, ...props }: any) {
    const isInline = !className;
    if (isInline) {
      return (
        <code className={className} {...props}>
          {children}
        </code>
      );
    }
    return <CodeBlock className={className}>{children}</CodeBlock>;
  },
  pre({ children }: any) {
    return <>{children}</>;
  },
};

/**
 * Parse legacy messages that have think blocks baked into content.
 * Handles both "<think>...</think>" and "Thinking Process:...\n</think>" formats.
 */
function parseThinkFromContent(content: string, existingThinking?: string): { thinking: string; content: string } {
  if (existingThinking || !content) return { thinking: existingThinking || "", content };

  const trimmed = content.trimStart();

  // Format: <think>...</think>
  if (trimmed.startsWith("<think>")) {
    const closeIdx = trimmed.indexOf("</think>");
    if (closeIdx !== -1) {
      const thinking = trimmed.slice(7, closeIdx).trim();
      const rest = trimmed.slice(closeIdx + 8).replace(/^\n+/, "");
      return { thinking, content: rest };
    }
  }

  // Format: Thinking Process:...\n</think>
  if (trimmed.startsWith("Thinking Process:") || trimmed.startsWith("Thinking Process\n")) {
    const closeIdx = trimmed.indexOf("</think>");
    if (closeIdx !== -1) {
      const thinkStart = trimmed.indexOf(":") !== -1 && trimmed.indexOf(":") < 20
        ? trimmed.indexOf(":") + 1
        : trimmed.indexOf("\n") + 1;
      const thinking = trimmed.slice(thinkStart, closeIdx).trim();
      const rest = trimmed.slice(closeIdx + 8).replace(/^\n+/, "");
      return { thinking, content: rest };
    }
  }

  return { thinking: "", content };
}

export function ChatMessage({ message }: { message: Message }) {
  const isUser = message.role === "user";

  // Parse think blocks from legacy stored messages
  const parsed = !isUser && !message.streaming
    ? parseThinkFromContent(message.content, message.thinking)
    : { thinking: message.thinking || "", content: message.content };

  const displayContent = parsed.content;
  const displayThinking = parsed.thinking;

  const hasThinking = !isUser && displayThinking.length > 0;
  const isThinking = message.streaming && !message.content && !!message.thinking;

  return (
    <div className={`message-animate py-5`}>
      <div className="max-w-3xl mx-auto px-6">
        <div className="flex gap-4">
          {/* Avatar */}
          <div
            className={`shrink-0 w-7 h-7 rounded-md flex items-center justify-center mt-0.5 ${
              isUser
                ? "bg-accent-purple/15 border border-accent-purple/25"
                : "bg-accent-green/15 border border-accent-green/25"
            }`}
          >
            {isUser ? (
              <User size={13} className="text-accent-purple" />
            ) : (
              <Bot size={13} className="text-accent-green" />
            )}
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1.5">
              <span className="text-xs font-mono text-text-tertiary uppercase tracking-wider">
                {isUser ? "You" : "DGInf"}
              </span>
              {message.trust && <TrustBadge trust={message.trust} />}
            </div>

            {/* Thinking block */}
            {hasThinking && (
              <ThinkingBlock
                thinking={displayThinking}
                streaming={isThinking}
              />
            )}

            {/* Verification panel — shown after streaming completes */}
            {message.trust && !message.streaming && (
              <div className="mb-3">
                <VerificationPanel trust={message.trust} />
              </div>
            )}

            {/* Main content */}
            <div
              className={`prose text-text-primary text-[15px] leading-relaxed ${
                message.streaming && !isThinking ? "streaming-cursor" : ""
              }`}
            >
              {displayContent ? (
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={markdownComponents}
                >
                  {displayContent}
                </ReactMarkdown>
              ) : message.streaming && !hasThinking ? (
                <span className="text-text-tertiary text-sm streaming-cursor" />
              ) : null}
            </div>

            {/* Live streaming metrics */}
            {!isUser && (message.streaming || message.tps) && (
              <StreamMetrics
                tps={message.tps}
                ttft={message.ttft}
                tokenCount={message.tokenCount}
                streaming={message.streaming}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
