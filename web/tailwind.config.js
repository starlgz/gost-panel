/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        bg: '#0f1115',
        panel: '#171a21',
        card: '#1c2029',
        edge: '#2c3340',
        input: '#12151b',
        txt: '#e6e8eb',
        muted: '#8a94a3',
      },
    },
  },
  plugins: [],
}
