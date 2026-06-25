import { forwardRef } from "react";

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  id: string;
  helper?: string;
  error?: string;
  accent?: "pink" | "teal";
}

const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, id, helper, error, accent = "pink", className = "", ...props },
  ref
) {
  const accentClass = accent === "teal" ? "input-teal" : "";
  const errorClass = error ? "input-error" : "";

  return (
    <div>
      <label
        htmlFor={id}
        className="block text-xs font-bold mb-1.5 uppercase tracking-wider"
        style={{ color: "#6B6052" }}
      >
        {label}
        {props.required && (
          <span className="ml-0.5" style={{ color: "#FF2D78" }}>*</span>
        )}
      </label>
      <input
        ref={ref}
        id={id}
        className={`input-zap ${accentClass} ${errorClass} ${className}`}
        {...props}
      />
      {helper && !error && (
        <p className="mt-1 text-xs" style={{ color: "#9CA3AF" }}>{helper}</p>
      )}
      {error && (
        <p className="mt-1 text-xs" style={{ color: "#e0245f" }} role="alert">
          {error}
        </p>
      )}
    </div>
  );
});

export default Input;
