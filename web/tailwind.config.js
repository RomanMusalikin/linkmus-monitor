/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
      },
      colors: {
        // Добавляем кастомный цвет для hover-эффектов карточек (между slate-700 и 800)
        slate: {
          750: '#293548',
        }
      }
    },
  },
  plugins: [],
}