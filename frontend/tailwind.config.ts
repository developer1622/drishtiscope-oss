import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: 'class',
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        bg: 'var(--color-bg)',
        panel: 'var(--color-panel)',
        panel2: 'var(--color-panel2)',
        border: 'var(--color-border)',
        txt: 'var(--color-txt)',
        muted: 'var(--color-muted)',
        cyan: 'var(--color-cyan)',
        green: 'var(--color-green)',
        amber: 'var(--color-amber)',
        rose: 'var(--color-rose)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      spacing: {
        '8': '8px',
      },
      borderRadius: {
        'xl': '12px',
      },
    },
  },
  plugins: [],
};

export default config;
