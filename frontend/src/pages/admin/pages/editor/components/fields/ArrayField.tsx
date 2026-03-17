import { useMemo, useCallback } from "react";
import type { FieldProps } from "./types";
import type { FieldSchema } from "@/theme/types";
import FieldRenderer from "../FieldRenderer";
import { useDragSort } from "../../hooks/useDragSort";

export default function ArrayField({ schema, value, onChange }: FieldProps) {
  const items: Record<string, unknown>[] = useMemo(
    () => (Array.isArray(value) ? (value as Record<string, unknown>[]) : []),
    [value],
  );
  const itemSchema: FieldSchema[] = schema.itemSchema ?? [];

  const moveItem = useCallback(
    (from: number, to: number) => {
      const next = [...items];
      const [moved] = next.splice(from, 1);
      next.splice(to, 0, moved);
      onChange(next);
    },
    [items, onChange],
  );

  const { makeDragHandlers } = useDragSort(moveItem);

  const getSummary = (item: Record<string, unknown>): string => {
    for (const field of itemSchema) {
      if (field.hidden) continue;
      const v = item[field.key];
      if (typeof v === "string" && v) return v;
      if (v && typeof v === "object" && "zh" in v && (v as { zh: string }).zh)
        return (v as { zh: string }).zh;
    }
    return "";
  };

  const handleFieldChange = (index: number, key: string, fieldValue: unknown) => {
    const next = items.map((item, i) =>
      i === index ? { ...item, [key]: fieldValue } : item
    );
    onChange(next);
  };

  const addItem = () => {
    const newItem: Record<string, unknown> = {
      _key: crypto.randomUUID(), // Stable React key for all array items
    };
    for (const field of itemSchema) {
      if (field.hidden && field.key === "id") {
        newItem[field.key] = crypto.randomUUID();
      } else if (field.defaultValue !== undefined) {
        newItem[field.key] = field.defaultValue;
      }
    }
    onChange([...items, newItem]);
  };

  const removeItem = (index: number) => {
    onChange(items.filter((_, i) => i !== index));
  };

  const moveUp = (index: number) => {
    if (index <= 0) return;
    const next = [...items];
    [next[index - 1], next[index]] = [next[index], next[index - 1]];
    onChange(next);
  };

  const moveDown = (index: number) => {
    if (index >= items.length - 1) return;
    const next = [...items];
    [next[index], next[index + 1]] = [next[index + 1], next[index]];
    onChange(next);
  };

  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">
        {schema.label}
      </label>
      {items.map((item, index) => {
        const summary = getSummary(item);
        const itemKey = (item.id as string) ?? (item._key as string) ?? String(index);
        return (
          <div
            key={itemKey}
            draggable
            {...makeDragHandlers(index)}
            className="border border-gray-200 rounded-lg p-3 mb-2 cursor-grab"
          >
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-gray-700">
                {schema.label} {index + 1}
                {summary && (
                  <span className="ml-2 text-gray-400 font-normal">
                    {summary}
                  </span>
                )}
              </span>
              <div className="flex gap-1">
                <button
                  type="button"
                  title="上移"
                  className="px-1 text-gray-400 hover:text-gray-600 text-sm"
                  onClick={() => moveUp(index)}
                >
                  ▲
                </button>
                <button
                  type="button"
                  title="下移"
                  className="px-1 text-gray-400 hover:text-gray-600 text-sm"
                  onClick={() => moveDown(index)}
                >
                  ▼
                </button>
                <button
                  type="button"
                  title="删除"
                  className="px-1 text-red-400 hover:text-red-600 text-sm"
                  onClick={() => removeItem(index)}
                >
                  ×
                </button>
              </div>
            </div>
            <div className="space-y-3">
              {itemSchema.map((field) => (
                <FieldRenderer
                  key={field.key}
                  schema={field}
                  value={item[field.key]}
                  onChange={(val) => handleFieldChange(index, field.key, val)}
                />
              ))}
            </div>
          </div>
        );
      })}
      <button
        type="button"
        className="mt-1 text-sm text-blue-600 hover:text-blue-800"
        onClick={addItem}
      >
        + 添加
      </button>
    </div>
  );
}
