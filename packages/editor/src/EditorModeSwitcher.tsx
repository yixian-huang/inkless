interface EditorModeSwitcherProps {
  mode: "richtext" | "markdown";
  onModeChange: (mode: "richtext" | "markdown") => void;
  /** Optional prefetch of the inactive mode's chunk on hover. */
  onPrefetch?: (mode: "richtext" | "markdown") => void;
}

export default function EditorModeSwitcher({
  mode,
  onModeChange,
  onPrefetch,
}: EditorModeSwitcherProps) {
  return (
    <div className="flex items-center gap-1 bg-slate-100 rounded-lg p-0.5">
      <button
        type="button"
        onClick={() => onModeChange("richtext")}
        onMouseEnter={() => onPrefetch?.("richtext")}
        onFocus={() => onPrefetch?.("richtext")}
        className={`px-3 py-1 text-xs rounded-xl transition ${
          mode === "richtext" ? "bg-white shadow text-slate-900" : "text-slate-500 hover:text-slate-700"
        }`}
      >
        Rich Text
      </button>
      <button
        type="button"
        onClick={() => onModeChange("markdown")}
        onMouseEnter={() => onPrefetch?.("markdown")}
        onFocus={() => onPrefetch?.("markdown")}
        className={`px-3 py-1 text-xs rounded-xl transition ${
          mode === "markdown" ? "bg-white shadow text-slate-900" : "text-slate-500 hover:text-slate-700"
        }`}
      >
        Markdown
      </button>
    </div>
  );
}
