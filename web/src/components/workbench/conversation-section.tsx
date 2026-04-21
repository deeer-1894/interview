import { Archive, ArrowUpRight, Copy, MoreHorizontal, PencilLine, Pin, Trash2 } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";

import type { Conversation } from "@/lib/types";

type ConversationStatusMeta = {
  label: string;
  tone: "idle" | "running" | "waiting" | "failed" | "done" | "archived";
  detail?: string;
  secondaryDetail?: string;
  highlights?: Array<{
    label: string;
    tone?: "neutral" | "info" | "warning" | "success";
  }>;
};

type ConversationHighlightTone = "neutral" | "info" | "warning" | "success";

type ConversationSectionProps = {
  title: string;
  conversations: Conversation[];
  selectedConversationId: string | null;
  openConversationMenuId: string | null;
  editingConversationId: string | null;
  editingConversationTitle: string;
  onEditingConversationTitleChange: (value: string) => void;
  onSelectConversation: (id: string) => void;
  onToggleMenu: Dispatch<SetStateAction<string | null>>;
  onCopyConversationTitle: (title: string) => Promise<void>;
  onStartRenameConversation: (conversation: Conversation) => void;
  onCancelRenameConversation: () => void;
  onSubmitRenameConversation: (conversation: Conversation) => Promise<void>;
  onToggleConversationPin: (conversation: Conversation) => Promise<void>;
  onToggleConversationArchive: (conversation: Conversation) => Promise<void>;
  onDeleteConversation: (conversation: Conversation) => Promise<void>;
  describeStatus: (conversation: Conversation) => ConversationStatusMeta;
};

export function ConversationSection({
  title,
  conversations,
  selectedConversationId,
  openConversationMenuId,
  editingConversationId,
  editingConversationTitle,
  onEditingConversationTitleChange,
  onSelectConversation,
  onToggleMenu,
  onCopyConversationTitle,
  onStartRenameConversation,
  onCancelRenameConversation,
  onSubmitRenameConversation,
  onToggleConversationPin,
  onToggleConversationArchive,
  onDeleteConversation,
  describeStatus,
}: ConversationSectionProps) {
  const highlightToneClass = (tone: ConversationHighlightTone = "neutral") => {
    switch (tone) {
      case "info":
        return "border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.92)] text-[rgb(29,78,216)]";
      case "warning":
        return "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]";
      case "success":
        return "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]";
      default:
        return "border-[rgba(226,231,239,0.96)] bg-[rgba(248,250,252,0.98)] text-[rgba(71,85,105,0.82)]";
    }
  };

  if (conversations.length === 0) {
    return null;
  }

  return (
    <div>
      <p className="px-2 text-[0.68rem] uppercase tracking-[0.18em] text-[rgba(107,114,128,0.62)]">{title}</p>
      <div className="mt-2 space-y-1">
        {conversations.map((conversation) => {
          const isEditing = editingConversationId === conversation.id;
          const isSelected = selectedConversationId === conversation.id;
          const isMenuOpen = openConversationMenuId === conversation.id;
          const status = describeStatus(conversation);
          const visibleHighlights = status.highlights?.slice(0, 2) ?? [];
          const hiddenHighlightLabels = status.highlights?.slice(2).map((highlight) => highlight.label) ?? [];
          const hiddenHighlights = Math.max(0, (status.highlights?.length ?? 0) - visibleHighlights.length);
          const statusToneClass =
            status.tone === "running"
              ? "border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] text-[rgb(37,99,235)]"
              : status.tone === "waiting"
                ? "border-[rgba(254,240,138,0.88)] bg-[rgba(254,252,232,0.98)] text-[rgb(161,98,7)]"
                : status.tone === "failed"
                  ? "border-[rgba(254,202,202,0.94)] bg-[rgba(254,242,242,0.98)] text-[rgb(185,28,28)]"
                  : status.tone === "done"
                    ? "border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.98)] text-[rgb(21,128,61)]"
                    : status.tone === "archived"
                      ? "border-[rgba(226,232,240,0.96)] bg-[rgba(248,250,252,0.98)] text-[rgba(71,85,105,0.82)]"
                      : "border-[rgba(226,231,239,0.96)] bg-[rgba(248,250,252,0.98)] text-[rgba(71,85,105,0.82)]";

          return (
            <div
              key={conversation.id}
              className={`group relative rounded-[1rem] border transition-all ${
                isSelected
                  ? "border-[rgba(191,219,254,0.82)] bg-[rgba(248,250,252,0.92)] shadow-[0_10px_24px_rgba(148,163,184,0.08)]"
                  : "border-transparent bg-transparent hover:border-[rgba(226,231,239,0.86)] hover:bg-[rgba(248,250,252,0.82)]"
              }`}
            >
              {isSelected ? (
                <div className="pointer-events-none absolute bottom-1 left-0 top-1 w-[2px] rounded-full bg-[linear-gradient(180deg,rgba(8,47,73,0.86),rgba(14,116,144,0.58))]" />
              ) : null}
              <button
                type="button"
                onClick={() => !isEditing && onSelectConversation(conversation.id)}
                className="flex w-full items-start justify-between gap-3 px-3 py-2 text-left"
              >
                <div className="min-w-0 flex-1">
                  {isEditing ? (
                    <div className="pr-10">
                      <input
                        autoFocus
                        value={editingConversationTitle}
                        onChange={(event) => onEditingConversationTitleChange(event.target.value)}
                        onClick={(event) => event.stopPropagation()}
                        onKeyDown={(event) => {
                          if (event.key === "Enter") {
                            event.preventDefault();
                            void onSubmitRenameConversation(conversation);
                          }
                          if (event.key === "Escape") {
                            event.preventDefault();
                            onCancelRenameConversation();
                          }
                        }}
                        onBlur={() => void onSubmitRenameConversation(conversation)}
                        className="w-full rounded-lg border border-[rgba(209,213,219,0.96)] bg-white px-3 py-1.5 text-[0.95rem] leading-5 text-[rgb(17,24,39)] outline-none focus:border-[rgba(59,130,246,0.28)]"
                      />
                    </div>
                  ) : (
                    <div className="flex items-start justify-between gap-2">
                      <p className="line-clamp-1 pr-2 font-display text-[0.93rem] leading-5 text-[rgb(17,24,39)]">{conversation.title}</p>
                      <span className="shrink-0 pt-0.5 text-[0.6rem] uppercase tracking-[0.12em] text-[rgba(107,114,128,0.6)]">
                        {new Date(conversation.updatedAt).toLocaleDateString()}
                      </span>
                    </div>
                  )}
                  <div className="mt-1.5 flex flex-wrap items-center gap-1">
                    <span className={`status-badge inline-flex items-center gap-1 px-2 py-0.5 text-[0.58rem] font-medium ${statusToneClass}`}>
                      <span className="h-1.5 w-1.5 rounded-full bg-current opacity-65" />
                      {status.label}
                    </span>
                    {visibleHighlights.map((highlight) => (
                      <span
                        key={`${conversation.id}-${highlight.label}`}
                        className={`status-badge px-2 py-0.5 text-[0.58rem] ${highlightToneClass(highlight.tone)}`}
                      >
                        {highlight.label}
                      </span>
                    ))}
                    {hiddenHighlights > 0 ? (
                      <span
                        className="group/hidden relative inline-flex"
                        title={hiddenHighlightLabels.join(" / ")}
                        aria-label={`更多标签：${hiddenHighlightLabels.join("，")}`}
                      >
                        <span className="status-badge border-[rgba(226,231,239,0.9)] bg-[rgba(255,255,255,0.92)] px-2 py-0.5 text-[0.58rem] text-[rgba(100,116,139,0.82)]">
                          +{hiddenHighlights}
                        </span>
                        <span className="pointer-events-none absolute left-1/2 top-[calc(100%+8px)] z-20 hidden min-w-[8rem] -translate-x-1/2 rounded-[0.8rem] border border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.98)] px-2.5 py-2 text-[10.5px] leading-4 text-[rgba(51,65,85,0.9)] shadow-[0_12px_24px_rgba(15,23,42,0.12)] group-hover/hidden:block">
                          {hiddenHighlightLabels.map((label) => (
                            <span key={`${conversation.id}-${label}-tooltip`} className="block whitespace-nowrap">
                              {label}
                            </span>
                          ))}
                        </span>
                      </span>
                    ) : null}
                    {conversation.pinned ? (
                      <span className="status-badge border-transparent bg-[rgba(241,245,249,0.98)] px-2 py-0.5 text-[0.58rem] text-[rgba(71,85,105,0.82)]">置顶</span>
                    ) : null}
                    {conversation.archived ? (
                      <span className="status-badge border-transparent bg-[rgba(248,250,252,0.98)] px-2 py-0.5 text-[0.58rem] text-[rgba(100,116,139,0.82)]">归档</span>
                    ) : null}
                  </div>
                  {status.detail ? <p className="mt-1.5 line-clamp-2 text-[11px] leading-4.5 text-[rgba(100,116,139,0.82)]">{status.detail}</p> : null}
                  {status.secondaryDetail ? (
                    <p className="mt-1 line-clamp-1 text-[10.5px] leading-4 text-[rgba(148,163,184,0.96)]">{status.secondaryDetail}</p>
                  ) : null}
                </div>
              </button>
              {!isEditing ? (
                <button
                  type="button"
                  aria-label="打开会话菜单"
                  onClick={(event) => {
                    event.stopPropagation();
                    onToggleMenu((current) => (current === conversation.id ? null : conversation.id));
                  }}
                  className={`absolute right-2.5 top-2.5 inline-flex h-7 w-7 items-center justify-center rounded-[0.82rem] border text-[rgba(100,116,139,0.76)] transition-all ${
                    isMenuOpen
                      ? "border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.98)] shadow-[0_6px_16px_rgba(15,23,42,0.08)]"
                      : "border-transparent bg-transparent opacity-0 group-hover:opacity-100 hover:border-[rgba(226,231,239,0.92)] hover:bg-[rgba(255,255,255,0.92)] hover:text-[rgba(71,85,105,0.88)]"
                  }`}
                >
                  <MoreHorizontal className="h-3.5 w-3.5" />
                </button>
              ) : null}
              {isMenuOpen ? (
                <div
                  className="absolute right-2 top-10 z-30 w-52 rounded-[1.05rem] border border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.98)] p-1.5 shadow-[0_16px_30px_rgba(15,23,42,0.1)] backdrop-blur-xl"
                  onClick={(event) => event.stopPropagation()}
                >
                  <button
                    type="button"
                    onClick={() => {
                      onSelectConversation(conversation.id);
                      onToggleMenu(null);
                    }}
                    className="flex w-full items-center gap-2.5 rounded-[0.8rem] px-3 py-2.5 text-left text-[13px] text-[rgb(30,41,59)] transition hover:bg-[rgba(248,250,252,0.98)]"
                  >
                    <ArrowUpRight className="h-3.5 w-3.5 text-[rgba(100,116,139,0.84)]" />
                    打开会话
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      void onCopyConversationTitle(conversation.title);
                      onToggleMenu(null);
                    }}
                    className="flex w-full items-center gap-2.5 rounded-[0.8rem] px-3 py-2.5 text-left text-[13px] text-[rgb(30,41,59)] transition hover:bg-[rgba(248,250,252,0.98)]"
                  >
                    <Copy className="h-3.5 w-3.5 text-[rgba(100,116,139,0.84)]" />
                    复制标题
                  </button>
                  <div className="my-1.5 h-px bg-[rgba(241,245,249,0.96)]" />
                  <button
                    type="button"
                    onClick={() => onStartRenameConversation(conversation)}
                    className="flex w-full items-center gap-2.5 rounded-[0.8rem] px-3 py-2.5 text-left text-[13px] text-[rgb(30,41,59)] transition hover:bg-[rgba(248,250,252,0.98)]"
                  >
                    <PencilLine className="h-3.5 w-3.5 text-[rgba(100,116,139,0.84)]" />
                    重命名
                  </button>
                  <button
                    type="button"
                    onClick={() => void onToggleConversationPin(conversation)}
                    className="flex w-full items-center gap-2.5 rounded-[0.8rem] px-3 py-2.5 text-left text-[13px] text-[rgb(30,41,59)] transition hover:bg-[rgba(248,250,252,0.98)]"
                  >
                    <Pin className="h-3.5 w-3.5 text-[rgba(100,116,139,0.84)]" />
                    {conversation.pinned ? "取消置顶" : "置顶聊天"}
                  </button>
                  <button
                    type="button"
                    onClick={() => void onToggleConversationArchive(conversation)}
                    className="flex w-full items-center gap-2.5 rounded-[0.8rem] px-3 py-2.5 text-left text-[13px] text-[rgb(30,41,59)] transition hover:bg-[rgba(248,250,252,0.98)]"
                  >
                    <Archive className="h-3.5 w-3.5 text-[rgba(100,116,139,0.84)]" />
                    {conversation.archived ? "取消归档" : "归档"}
                  </button>
                  <button
                    type="button"
                    onClick={() => void onDeleteConversation(conversation)}
                    className="mt-1 flex w-full items-center gap-2.5 rounded-[0.8rem] px-3 py-2.5 text-left text-[13px] text-[rgb(220,38,38)] transition hover:bg-[rgba(254,242,242,0.98)]"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    删除
                  </button>
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
