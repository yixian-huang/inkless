import { useState, useRef, useEffect } from "react";

interface DropdownOption {
  label: string;
  value: string;
  active?: boolean;
}

interface ToolbarDropdownProps {
  label: string;
  options: DropdownOption[];
  onSelect: (value: string) => void;
  title?: string;
}

export default function ToolbarDropdown({ label, options, onSelect, title }: ToolbarDropdownProps) {
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
        className="px-2 py-1 text-sm rounded transition-colors text-gray-600 hover:bg-gray-200 hover:text-gray-900 flex items-center gap-0.5"
      >
        <span className="text-xs">{label}</span>
        <span className="text-[10px]">▾</span>
      </button>
      {open && (
        <div className="absolute top-full left-0 mt-1 py-1 bg-white rounded-lg shadow-lg border border-gray-200 z-50 min-w-[120px] max-h-60 overflow-y-auto">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => { onSelect(opt.value); setOpen(false); }}
              className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 ${
                opt.active ? "bg-blue-50 text-blue-700 font-medium" : "text-gray-700"
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
