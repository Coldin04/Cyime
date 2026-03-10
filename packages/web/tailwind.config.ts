import type { Config } from "tailwindcss";

export default {
  darkMode: "media",
  content: ["./src/**/*.{html,js,svelte,ts}", "./node_modules/@milkdown/**/*.{js,ts,css}"],

  theme: {
    extend: {
      colors: {
        'riptide': {
            '50': '#f0fdfb',
            '100': '#cbfcf4',
            '200': '#77f6e3',
            '300': '#5beddc',
            '400': '#29d8ca',
            '500': '#10bcb1',
            '600': '#0a9791',
            '700': '#0d7875',
            '800': '#0f605f',
            '900': '#124f4e',
            '950': '#032f30',
        },

      }
    }
  },

  plugins: [require("@tailwindcss/typography")]
} as Config;
