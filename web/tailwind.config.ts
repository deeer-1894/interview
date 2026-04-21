import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        card: "hsl(var(--card))",
        "card-foreground": "hsl(var(--card-foreground))",
        popover: "hsl(var(--popover))",
        "popover-foreground": "hsl(var(--popover-foreground))",
        primary: "hsl(var(--primary))",
        "primary-foreground": "hsl(var(--primary-foreground))",
        secondary: "hsl(var(--secondary))",
        "secondary-foreground": "hsl(var(--secondary-foreground))",
        muted: "hsl(var(--muted))",
        "muted-foreground": "hsl(var(--muted-foreground))",
        accent: "hsl(var(--accent))",
        "accent-foreground": "hsl(var(--accent-foreground))",
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 6px)",
      },
      boxShadow: {
        paper: "0 14px 36px rgba(20, 34, 66, 0.06)",
        spotlight: "0 0 0 1px rgba(255,255,255,0.55), 0 22px 72px rgba(61, 93, 158, 0.1)",
      },
      fontFamily: {
        display: ["'Sora'", "'Noto Sans SC'", "sans-serif"],
        body: ["'Noto Sans SC'", "sans-serif"],
        serif: ["'Sora'", "'Noto Sans SC'", "sans-serif"],
      },
      backgroundImage: {
        grid: "linear-gradient(to right, rgba(15,18,10,0.08) 1px, transparent 1px), linear-gradient(to bottom, rgba(15,18,10,0.08) 1px, transparent 1px)",
      },
      animation: {
        "float-slow": "floatSlow 16s ease-in-out infinite",
        "pulse-line": "pulseLine 3s ease-in-out infinite",
        reveal: "reveal 0.9s cubic-bezier(.2,.8,.2,1) both",
      },
      keyframes: {
        floatSlow: {
          "0%, 100%": { transform: "translate3d(0, 0, 0)" },
          "50%": { transform: "translate3d(0, -14px, 0)" },
        },
        pulseLine: {
          "0%, 100%": { opacity: "0.45" },
          "50%": { opacity: "1" },
        },
        reveal: {
          from: { opacity: "0", transform: "translateY(24px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
      },
    },
  },
  plugins: [],
};

export default config;
