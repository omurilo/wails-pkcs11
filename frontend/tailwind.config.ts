/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
 theme: {
    extend: {
      colors: {
        // Exemplo: Cores primárias e secundárias personalizadas
        primary: {
          50: '#e0f2fe',
          100: '#bae6fd',
          200: '#7dd3fc',
          300: '#38bdf8',
          400: '#0ea5e9',
          500: '#017acb', // Nosso azul principal
          600: '#0264a8',
          700: '#034e85',
          800: '#053961',
          900: '#06243e',
          950: '#03172c',
        },
        secondary: { // Para um contraste sutil
          50: '#f0f4f8',
          100: '#e1e7ee',
          200: '#c5d0db',
          300: '#a8b8c7',
          400: '#8c9ead',
          500: '#708398', // Nosso cinza escuro principal
          600: '#586b81',
          700: '#405267',
          800: '#2a3b4d',
          900: '#172431',
          950: '#0c1218',
        },
        // Mantenha outras cores como red, green, yellow, etc. se forem usadas
      },
    },
  },
  plugins: [],
}
