import type { FieldProps } from "./types";

export default function SelectField({ schema, value, onChange }: FieldProps) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">
        {schema.label}
      </label>
      <select
        className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
        value={value != null ? String(value) : ""}
        onChange={(e) => {
          const v = e.target.value;
          if (v === "") {
            onChange(undefined);
          } else {
            // Per-option coercion: find the matching option and use its original typed value
            const selectedOpt = schema.options?.find((opt) => String(opt.value) === v);
            onChange(selectedOpt ? selectedOpt.value : v);
          }
        }}
      >
        <option value="">请选择</option>
        {schema.options?.map((opt) => (
          <option key={String(opt.value)} value={String(opt.value)}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}
