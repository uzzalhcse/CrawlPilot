/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}'
  ],
  safelist: [
    // Border colors
    'border-blue-500',
    'border-purple-500',
    'border-pink-500',
    'border-green-500',
    'border-yellow-500',
    'border-cyan-500',
    'border-orange-500',
    'border-indigo-500',
    'border-gray-500',
    // Background colors
    'bg-blue-500/15',
    'bg-purple-500/15',
    'bg-pink-500/15',
    'bg-green-500/15',
    'bg-yellow-500/15',
    'bg-cyan-500/15',
    'bg-orange-500/15',
    'bg-indigo-500/15',
    'bg-gray-500/15',
    // Badge colors
    'bg-blue-600',
    'bg-purple-600',
    'bg-pink-600',
    'bg-green-600',
    'bg-yellow-600',
    'bg-cyan-600',
    'bg-orange-600',
    'bg-indigo-600',
    'bg-gray-600',
    // Border left colors for field cards
    '!border-l-blue-500',
    '!border-l-purple-500',
    '!border-l-pink-500',
  ],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))'
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))'
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))'
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))'
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))'
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))'
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))'
        }
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)'
      },
      keyframes: {
        'crawlify-pulse': {
          '0%, 100%': { 
            boxShadow: '0 0 0 1px rgba(59, 130, 246, 0.3), 0 4px 12px rgba(59, 130, 246, 0.2)' 
          },
          '50%': { 
            boxShadow: '0 0 0 2px rgba(59, 130, 246, 0.5), 0 6px 16px rgba(59, 130, 246, 0.3)' 
          }
        },
        'crawlify-success-pulse': {
          '0%': { transform: 'scale(1)' },
          '50%': { transform: 'scale(1.02)' },
          '100%': { transform: 'scale(1)' }
        }
      },
      animation: {
        'crawlify-pulse': 'crawlify-pulse 2s ease-in-out infinite',
        'crawlify-success-pulse': 'crawlify-success-pulse 0.5s ease-out'
      }
    }
  },
  plugins: []
}
