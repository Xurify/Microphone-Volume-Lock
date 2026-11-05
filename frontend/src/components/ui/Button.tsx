import React from 'react';

interface ButtonProps {
  onClick: () => void;
  children: React.ReactNode;
  variant?: 'primary' | 'secondary';
  disabled?: boolean;
  className?: string;
}

const baseClasses = "inline-flex h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-semibold ring-1 ring-inset transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-8)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--color-background)]";

const variantClasses = {
  primary: "bg-[var(--accent-9)] text-[var(--accent-contrast)] ring-[var(--accent-a8)] shadow-[inset_0_1px_0_rgba(255,255,255,0.15)] enabled:hover:bg-[var(--accent-10)]",
  secondary: "bg-[var(--gray-a3)] text-[var(--gray-12)] ring-[var(--gray-a5)] enabled:hover:bg-[var(--gray-a4)]",
};

const Button: React.FC<ButtonProps> = ({
  onClick,
  children,
  variant = 'primary',
  disabled = false,
  className = ''
}) => {
  const disabledClasses = disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer";

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`${baseClasses} ${variantClasses[variant]} ${disabledClasses} ${className}`}
    >
      {children}
    </button>
  );
};

export default Button;
