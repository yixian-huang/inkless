import { useState } from "react";

interface MetadataEditorProps {
  value: Record<string, unknown>;
  onChange: (value: Record<string, unknown>) => void;
}

export default function MetadataEditor({ value, onChange }: MetadataEditorProps) {
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  const entries = Object.entries(value || {});

  const handleAdd = () => {
    if (!newKey.trim()) return;
    onChange({ ...value, [newKey.trim()]: newValue });
    setNewKey("");
    setNewValue("");
  };

  const handleRemove = (key: string) => {
    const next = { ...value };
    delete next[key];
    onChange(next);
  };

  const handleValueChange = (key: string, val: string) => {
    onChange({ ...value, [key]: val });
  };

  return (
    <div className="space-y-2">
      {entries.map(([key, val]) => (
        <div key={key} className="flex items-center gap-2">
          <input
            type="text"
            value={key}
            readOnly
            className="w-1/3 px-2 py-1.5 border border-gray-300 rounded text-sm bg-gray-50"
          />
          <input
            type="text"
            value={String(val ?? "")}
            onChange={(e) => handleValueChange(key, e.target.value)}
            className="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm"
          />
          <button
            type="button"
            onClick={() => handleRemove(key)}
            className="px-2 py-1.5 text-red-500 hover:text-red-700 text-sm"
          >
            ×
          </button>
        </div>
      ))}
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={newKey}
          onChange={(e) => setNewKey(e.target.value)}
          placeholder="Key"
          className="w-1/3 px-2 py-1.5 border border-gray-300 rounded text-sm"
          onKeyDown={(e) => e.key === "Enter" && handleAdd()}
        />
        <input
          type="text"
          value={newValue}
          onChange={(e) => setNewValue(e.target.value)}
          placeholder="Value"
          className="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm"
          onKeyDown={(e) => e.key === "Enter" && handleAdd()}
        />
        <button
          type="button"
          onClick={handleAdd}
          className="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 border border-gray-300 rounded text-sm"
        >
          Add
        </button>
      </div>
    </div>
  );
}
