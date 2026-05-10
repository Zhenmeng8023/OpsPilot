import type { InputHTMLAttributes, TextareaHTMLAttributes } from "react";

type TextFieldInputProps = {
  label?: string;
  multiline?: false;
} & InputHTMLAttributes<HTMLInputElement>;

type TextFieldTextareaProps = {
  label?: string;
  multiline: true;
} & TextareaHTMLAttributes<HTMLTextAreaElement>;

type TextFieldProps = TextFieldInputProps | TextFieldTextareaProps;

export function TextField(props: TextFieldProps) {
  if (props.multiline) {
    const { label, multiline: _multiline, ...textareaProps } = props;
    const field = <textarea {...textareaProps} />;
    if (!label) return field;
    return (
      <label>
        <span>{label}</span>
        {field}
      </label>
    );
  }

  const { label, multiline: _multiline, ...inputProps } = props;
  const field = <input {...inputProps} />;
  if (!label) return field;
  return (
    <label>
      <span>{label}</span>
      {field}
    </label>
  );
}
