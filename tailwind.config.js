// Tailwind config (the only JS in the repo — build config, not app code).
// Original jevai palette: warm paper, indigo brand, a green→amber→red risk scale.
/** @type {import('tailwindcss').Config} */
module.exports = {
  // scan .go too — some risk-color classes are built in helpers.go, not literal in templ
  content: ["./internal/web/**/*.templ", "./internal/web/**/*.go"],
  theme: {
    extend: {
      colors: {
        paper: "#FBFBF9",
        surface: "#FFFFFF",
        ink: "#17181B",
        line: "#E6E6E1",
        muted: "#6B6B66",
        brand: "#5B54E6",
        brandsoft: "#EDECFC",
        safe: "#10B981",
        watch: "#F59E0B",
        flag: "#EF4444",
      },
      fontFamily: {
        display: ["'Space Grotesk'", "system-ui", "sans-serif"],
        sans: ["Inter", "system-ui", "sans-serif"],
        mono: ["'JetBrains Mono'", "ui-monospace", "monospace"],
      },
    },
  },
  plugins: [],
};
