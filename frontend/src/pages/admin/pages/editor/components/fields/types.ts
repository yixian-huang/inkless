import type { FieldSchema } from "@/theme/types";

export type { FieldType, FieldSchema } from "@/theme/types";

export interface FieldProps {
  schema: FieldSchema;
  value: unknown;
  onChange: (value: unknown) => void;
}
