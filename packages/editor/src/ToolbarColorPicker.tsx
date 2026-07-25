import { useState, useRef, useEffect } from "react";

const PRESET_COLORS = [
  "#000000", "#434343", "#666666", "#999999",
  "#dc2626", "#ea580c", "#d97706", "#ca8a04",
  "#16a34a", "#059669", "#0891b2", "#2563eb",
  "#7c3aed", "#9333ea", "#db2777", "#e11d48",
];

interface ToolbarColorPickerProps {
  color?: string;
  onChange: (color: string) => void;
  onReset: () => void;
  icon: React.ReactNode;
  title: string;
}

export default function ToolbarColorPicker({ color, onChange, onReset, icon, title }: ToolbarColorPickerProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        title={title}
        className="px-2 py-1 text-sm rounded transition-colors text-slate-600 hover:bg-slate-200 hover:text-slate-900 flex items-center gap-1"
      >
        {icon}
        <span
          className="w-3 h-1.5 rounded-sm border border-slate-200"
          style={{ backgroundColor: color || "transparent" }}
        />
      </button>
      {open && (
        <div className="absolute top-full left-0 mt-1 p-2 bg-white rounded-lg shadow-lg border border-slate-200 z-50 w-[172px]">
          <div className="grid grid-cols-4 gap-1.5 mb-2">
            {PRESET_COLORS.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => { onChange(c); setOpen(false); }}
                className="w-8 h-8 rounded border border-slate-200 hover:scale-110 transition-transform"
                style={{ backgroundColor: c }}
                title={c}
              />
            ))}
          </div>
          <button
            type="button"
            onClick={() => { onReset(); setOpen(false); }}
            className="w-full text-xs text-slate-500 hover:text-slate-800 py-1 border-t border-slate-100"
          >
            重置
          </button>
        </div>
      )}
    </div>
  );
}
