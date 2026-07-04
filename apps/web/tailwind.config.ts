import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#17211b",
        moss: "#315846",
        mint: "#dff5e8",
        coral: "#f07b63",
        gold: "#d6a53a",
        sky: "#dceefa"
      },
      boxShadow: {
        panel: "0 18px 45px rgba(23, 33, 27, 0.08)"
      }
    }
  },
  plugins: []
};

export default config;
