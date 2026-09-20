// Tailwind config (the only JS in the repo — build config, not app code).
// Palette + type lifted from typesafe.ai: off-white paper, near-black ink,
// Host Grotesk display, Fragment Mono for labels, hot-pink accent.
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/web/**/*.templ"],
  theme: {
    extend: {
      colors: {
        paper: "#FEFEFE",
        ink: "#1E1E1E",
        line: "#E5E5E5",
        muted: "#858585",
        pink: "#F386A1",
        magenta: "#D45BB6",
        grass: "#03AA5C",
        teal: "#09AEA1",
      },
      fontFamily: {
        display: ["'Host Grotesk'", "Inter", "system-ui", "sans-serif"],
        sans: ["'Host Grotesk'", "Inter", "system-ui", "sans-serif"],
        mono: ["'Fragment Mono'", "'JetBrains Mono'", "ui-monospace", "monospace"],
      },
    },
  },
  plugins: [],
};
