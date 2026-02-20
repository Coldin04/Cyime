import type { Config } from "tailwindcss";

export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],

  theme: {
    extend: {
      colors: {
        "spring-green": {
          "50": "#edfff7",
          "100": "#d5ffef",
          "200": "#aeffdf",
          "300": "#70ffc7",
          "400": "#2bfda8",
          "500": "#00ec8d",
          "600": "#00c06e",
          "700": "#009659",
          "800": "#067549",
          "900": "#07603e",
          "950": "#003722"
        }
      }
    }
  },

  plugins: [require("@tailwindcss/typography")]
} as Config;
