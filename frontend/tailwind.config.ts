/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ["class"],
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        background: "hsl(240, 10%, 4%)",
        foreground: "hsl(0, 0%, 98%)",
        card: "hsl(240, 10%, 6%)",
        border: "hsl(240, 10%, 15%)",
        primary: "hsl(217, 91%, 60%)",
        destructive: "hsl(0, 84%, 60%)",
        warning: "hsl(38, 92%, 50%)",
        success: "hsl(142, 71%, 45%)",
      },
    },
  },
  plugins: [],
};
