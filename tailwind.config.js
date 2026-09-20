// Tailwind config (the only JS in the repo — build config, not app code).
// jevai identity: warm paper + ink, an acid-lime signature used as blocks/marker
// (not text — it's too light on white), and a green→amber→red risk scale.
/** @type {import('tailwindcss').Config} */
module.exports = {
  // scan .go too — some risk-color classes are built in helpers.go, not literal in templ
  content: ["./internal/web/**/*.templ", "./internal/web/**/*.go"],
  theme: {
    extend: {
      colors: {
        paper: "#FCFCF7",
        surface: "#FFFFFF",
        ink: "#14140F",
        line: "#E4E4DA",
        muted: "#6C6C61",
        lime: "#C6F24E",
        safe: "#16A34A",
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
