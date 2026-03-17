import DynamicForm from "./DynamicForm";
import { settingsSchema } from "@/theme/sectionSchemas";
import type { SectionSettings as SectionSettingsType } from "@/theme/types";

interface Props {
  settings: SectionSettingsType;
  onChange: (settings: SectionSettingsType) => void;
}

export default function SectionSettingsForm({ settings, onChange }: Props) {
  return (
    <details className="border-t border-gray-200 pt-3 mt-4">
      <summary className="text-xs font-semibold text-gray-600 cursor-pointer select-none">
        显示设置
      </summary>
      <div className="mt-3">
        <DynamicForm
          schema={settingsSchema}
          data={settings as Record<string, unknown>}
          onChange={(data) => onChange(data as SectionSettingsType)}
        />
      </div>
    </details>
  );
}
