import { useState } from "react";
import DynamicForm from "./DynamicForm";
import SectionSettingsForm from "./SectionSettings";
import { sectionSchemas } from "@/theme/sectionSchemas";
import type { SectionData, SectionSettings as SectionSettingsType } from "@/theme/types";

interface Props {
  section: SectionData;
  onDataChange: (data: Record<string, unknown>) => void;
  onSettingsChange: (settings: SectionSettingsType) => void;
}

export default function PropertiesPanel({ section, onDataChange, onSettingsChange }: Props) {
  const schema = sectionSchemas[section.type];
  const [jsonMode, setJsonMode] = useState(false);
  const [jsonText, setJsonText] = useState("");

  const switchToJson = () => {
    setJsonText(JSON.stringify(section.data, null, 2));
    setJsonMode(true);
  };

  const switchToForm = () => {
    try {
      const parsed = JSON.parse(jsonText);
      onDataChange(parsed);
      setJsonMode(false);
    } catch {
      // keep in JSON mode if invalid
    }
  };

  const showJsonEditor = !schema || jsonMode;

  return (
    <div>
      <div className="mb-3">
        <span className="text-xs text-gray-500">
          类型: {section.type}
          {section.variant ? ` / ${section.variant}` : ""}
          {section.locked ? " (锁定)" : ""}
        </span>
      </div>

      {showJsonEditor ? (
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">数据 (JSON)</label>
          {jsonMode ? (
            <textarea
              value={jsonText}
              onChange={(e) => setJsonText(e.target.value)}
              rows={12}
              className="w-full border border-gray-300 rounded-md px-2 py-1.5 text-xs font-mono resize-none"
              spellCheck={false}
            />
          ) : (
            <textarea
              value={JSON.stringify(section.data, null, 2)}
              onChange={(e) => {
                try { onDataChange(JSON.parse(e.target.value)); } catch { /* ignore parse errors while typing */ }
              }}
              rows={12}
              className="w-full border border-gray-300 rounded-md px-2 py-1.5 text-xs font-mono resize-none"
              spellCheck={false}
            />
          )}
        </div>
      ) : (
        <DynamicForm schema={schema} data={section.data} onChange={onDataChange} />
      )}

      <SectionSettingsForm
        settings={section.settings || {}}
        onChange={onSettingsChange}
      />

      {schema && (
        <div className="mt-3 text-center">
          <button
            type="button"
            onClick={jsonMode ? switchToForm : switchToJson}
            className="text-xs text-gray-400 hover:text-blue-500 underline"
          >
            {jsonMode ? "切换到表单编辑" : "切换到 JSON 编辑"}
          </button>
        </div>
      )}
    </div>
  );
}
