import React from 'react';

export function Logo({
  size = 28,
  className = '',
  showText = false,
}: {
  size?: number;
  className?: string;
  showText?: boolean;
}) {
  return (
    <div className={`flex items-center gap-2.5 select-none ${className}`}>
      {/* SVG Icon Emblem */}
      <svg
        width={size}
        height={size}
        viewBox="0 0 64 64"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="shrink-0 transition-transform duration-300 hover:scale-105"
      >
        <defs>
          <linearGradient id="logo-cyan" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#3ce0cf" />
            <stop offset="100%" stopColor="#00adb5" />
          </linearGradient>
          <linearGradient id="logo-purple" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#a855f7" />
            <stop offset="100%" stopColor="#6366f1" />
          </linearGradient>
          <linearGradient id="logo-green" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#3dd68c" />
            <stop offset="100%" stopColor="#10b981" />
          </linearGradient>
          <radialGradient id="logo-iris-glow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#3ce0cf" stopOpacity="1" />
            <stop offset="70%" stopColor="#00adb5" stopOpacity="0.8" />
            <stop offset="100%" stopColor="#07080d" stopOpacity="0" />
          </radialGradient>
        </defs>

        {/* Base Rounded Badge */}
        <rect width="64" height="64" rx="16" fill="var(--color-panel2, #0e1118)" />
        <rect width="64" height="64" rx="16" stroke="var(--color-border, #1e2430)" strokeWidth="1.5" />

        {/* Radar Ring */}
        <circle cx="32" cy="32" r="24" stroke="var(--color-border, #1e2430)" strokeWidth="1" strokeDasharray="2 3" />
        <circle cx="32" cy="32" r="18" stroke="url(#logo-green)" strokeWidth="1" strokeOpacity="0.4" strokeDasharray="4 2" />

        {/* Orbital Kernel Scope Rings */}
        <ellipse cx="32" cy="32" rx="23" ry="11" transform="rotate(-28 32 32)" stroke="url(#logo-cyan)" strokeWidth="1.6" strokeOpacity="0.9" />
        <ellipse cx="32" cy="32" rx="20" ry="8" transform="rotate(32 32 32)" stroke="url(#logo-purple)" strokeWidth="1.4" strokeOpacity="0.75" />

        {/* Orbital Nodes */}
        <circle cx="12" cy="22" r="1.5" fill="#3ce0cf" />
        <circle cx="52" cy="42" r="1.5" fill="#3dd68c" />
        <circle cx="48" cy="20" r="1.2" fill="#a855f7" />

        {/* All-Seeing Vision Eye ("Drishti") */}
        <path d="M16 32C20 23 44 23 48 32C44 41 20 41 16 32Z" stroke="url(#logo-cyan)" strokeWidth="2" strokeLinejoin="round" fill="var(--color-panel, #07080d)" fillOpacity="0.7" />

        {/* Inner Eye Contour */}
        <path d="M21 32C24 26 40 26 43 32C40 38 24 38 21 32Z" stroke="url(#logo-purple)" strokeWidth="1.2" strokeOpacity="0.8" />

        {/* Central Iris & Pupil */}
        <circle cx="32" cy="32" r="6" fill="#07080d" stroke="url(#logo-cyan)" strokeWidth="1.5" />
        <circle cx="32" cy="32" r="3.5" fill="url(#logo-iris-glow)" />
        <circle cx="33" cy="31" r="1.2" fill="#ffffff" />

        {/* Scope Crosshairs */}
        <line x1="32" y1="21" x2="32" y2="25" stroke="#3ce0cf" strokeWidth="1.2" strokeLinecap="round" />
        <line x1="32" y1="39" x2="32" y2="43" stroke="#3ce0cf" strokeWidth="1.2" strokeLinecap="round" />
        <line x1="21" y1="32" x2="25" y2="32" stroke="#3ce0cf" strokeWidth="1.2" strokeLinecap="round" />
        <line x1="39" y1="32" x2="43" y2="32" stroke="#3ce0cf" strokeWidth="1.2" strokeLinecap="round" />
      </svg>

      {/* Brand Text (Optional) */}
      {showText && (
        <div className="flex flex-col shrink-0">
          <div className="flex items-center gap-1.5 leading-none">
            <span className="text-txt font-bold tracking-tight text-base sm:text-lg">DrishtiScope</span>
            <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded bg-cyan/15 text-cyan border border-cyan/30">
              दृष्टि
            </span>
          </div>
          <span className="text-[10px] font-normal text-muted leading-tight">
            Real-Time Agentic AI & LLM Process Observability
          </span>
        </div>
      )}
    </div>
  );
}
