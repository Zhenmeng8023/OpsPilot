import type { SelectHTMLAttributes } from "react";

type SelectOption = {
  value: string;
  label: string;
};

type SelectFieldProps = SelectHTMLAttributes<HTMLSelectElement> & {
  label?: string;
  options: SelectOption[];
};

export function SelectField({ label, options, ...props }: SelectFieldProps) {
  const field = (
    <select {...props}>
      {options.map((option) => (
        <option key={`${option.value}-${option.label}`} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  );

  if (!label) return field;
  return (
    <label>
      <span>{label}</span>
      {field}
    </label>
  );
}
