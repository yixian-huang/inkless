import type { ThemeSettingGroup } from "@/plugins/types";

interface Props {
  schema: ThemeSettingGroup[];
  values: Record<string, any>;
  onChange: (values: Record<string, any>) => void;
}

export default function ThemeSettingsForm({ schema, values, onChange }: Props) {
  const getValue = (group: string, name: string, defaultValue?: any) => {
    const key = `${group}.${name}`;
    return values[key] !== undefined ? values[key] : defaultValue;
  };

  const setValue = (group: string, name: string, value: any) => {
    const key = `${group}.${name}`;
    onChange({ ...values, [key]: value });
  };

  return (
    <div>
      {schema.map((group) => (
        <div key={group.group} className="bg-white rounded-lg shadow p-6 mb-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">{group.labelZh}</h3>
          <div className="space-y-4">
            {group.fields.map((field) => {
              const val = getValue(group.group, field.name, field.defaultValue);
              return (
                <div key={field.name}>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    {field.labelZh}
                  </label>

                  {field.type === "text" && (
                    <input
                      type="text"
                      value={val ?? ""}
                      onChange={(e) => setValue(group.group, field.name, e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  )}

                  {field.type === "textarea" && (
                    <textarea
                      value={val ?? ""}
                      onChange={(e) => setValue(group.group, field.name, e.target.value)}
                      rows={3}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  )}

                  {field.type === "number" && (
                    <input
                      type="number"
                      value={val ?? 0}
                      onChange={(e) => setValue(group.group, field.name, Number(e.target.value))}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  )}

                  {field.type === "boolean" && (
                    <label className="inline-flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={!!val}
                        onChange={(e) => setValue(group.group, field.name, e.target.checked)}
                        className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-600">{field.labelZh}</span>
                    </label>
                  )}

                  {field.type === "select" && (
                    <select
                      value={val ?? ""}
                      onChange={(e) => setValue(group.group, field.name, e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    >
                      {field.options?.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                  )}

                  {field.type === "color" && (
                    <div className="flex items-center gap-3">
                      <input
                        type="color"
                        value={val ?? "#000000"}
                        onChange={(e) => setValue(group.group, field.name, e.target.value)}
                        className="w-10 h-10 rounded border border-gray-200 cursor-pointer"
                      />
                      <input
                        type="text"
                        value={val ?? ""}
                        onChange={(e) => setValue(group.group, field.name, e.target.value)}
                        className="flex-1 text-xs font-mono text-gray-500 border border-gray-200 rounded px-2 py-1"
                      />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}
