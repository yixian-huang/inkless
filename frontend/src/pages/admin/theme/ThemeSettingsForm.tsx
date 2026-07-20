import type { ThemeSettingGroup } from "@/plugins/types";
import {
  AdminCard,
  AdminCheckbox,
  AdminField,
  AdminInput,
  AdminSelect,
  AdminTextarea,
} from "@/components/admin/ui";

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
    <div className="space-y-4">
      {schema.map((group) => (
        <AdminCard key={group.group} title={group.labelZh}>
          <div className="space-y-4">
            {group.fields.map((field) => {
              const val = getValue(group.group, field.name, field.defaultValue);
              return (
                <AdminField key={field.name} label={field.type === "boolean" ? undefined : field.labelZh}>
                  {field.type === "text" && (
                    <AdminInput
                      type="text"
                      value={val ?? ""}
                      onChange={(e) => setValue(group.group, field.name, e.target.value)}
                    />
                  )}

                  {field.type === "textarea" && (
                    <AdminTextarea
                      value={val ?? ""}
                      onChange={(e) => setValue(group.group, field.name, e.target.value)}
                      rows={3}
                    />
                  )}

                  {field.type === "number" && (
                    <AdminInput
                      type="number"
                      value={val ?? 0}
                      onChange={(e) => setValue(group.group, field.name, Number(e.target.value))}
                    />
                  )}

                  {field.type === "boolean" && (
                    <AdminCheckbox
                      checked={!!val}
                      onChange={(e) => setValue(group.group, field.name, e.target.checked)}
                      label={field.labelZh}
                    />
                  )}

                  {field.type === "select" && (
                    <AdminSelect
                      value={val ?? ""}
                      onChange={(e) => setValue(group.group, field.name, e.target.value)}
                      className="w-full"
                    >
                      {field.options?.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </AdminSelect>
                  )}

                  {field.type === "color" && (
                    <div className="flex items-center gap-3">
                      <input
                        type="color"
                        value={val ?? "#000000"}
                        onChange={(e) => setValue(group.group, field.name, e.target.value)}
                        className="h-10 w-10 cursor-pointer rounded-xl border border-slate-200"
                      />
                      <AdminInput
                        type="text"
                        value={val ?? ""}
                        onChange={(e) => setValue(group.group, field.name, e.target.value)}
                        className="flex-1 font-mono text-xs"
                      />
                    </div>
                  )}
                </AdminField>
              );
            })}
          </div>
        </AdminCard>
      ))}
    </div>
  );
}
