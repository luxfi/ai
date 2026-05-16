interface LogoProps {
  size?: number;
  className?: string;
}

export function Logo({ size = 24, className = '' }: LogoProps) {
  // Lux AI logo - triangle with cyan accent
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Outer triangle - cyan */}
      <path
        d="M50 5 L95 90 L5 90 Z"
        fill="none"
        stroke="oklch(0.55 0.16 220)"
        strokeWidth="4"
      />
      {/* Inner triangle - filled */}
      <path
        d="M50 25 L75 75 L25 75 Z"
        fill="oklch(0.55 0.16 220)"
      />
      {/* AI circuit pattern */}
      <circle cx="50" cy="55" r="8" fill="currentColor" />
      <line x1="50" y1="47" x2="50" y2="35" stroke="currentColor" strokeWidth="2" />
      <line x1="42" y1="55" x2="30" y2="55" stroke="currentColor" strokeWidth="2" />
      <line x1="58" y1="55" x2="70" y2="55" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

export function LogoWithText({ size = 24 }: { size?: number }) {
  return (
    <div className="flex items-center gap-2">
      <Logo size={size} />
      <span className="font-bold text-lg">Lux AI</span>
    </div>
  );
}
