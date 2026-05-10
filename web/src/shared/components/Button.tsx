import type { ButtonHTMLAttributes } from "react";

type ButtonTone = "default" | "danger";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: ButtonTone;
};

export function Button({ tone = "default", className = "", type = "button", ...props }: ButtonProps) {
  const classes = [className];
  if (tone === "danger") {
    classes.push("danger-button");
  }
  return <button type={type} className={classes.filter(Boolean).join(" ")} {...props} />;
}
