# Crawlify Frontend

Modern Vue 3 dashboard for Crawlify web crawler with Shadcn Vue, Tailwind CSS, and Pinia.

## Tech Stack

- **Vue 3** - Progressive JavaScript framework
- **TypeScript** - Type safety
- **Vite** - Fast build tool
- **Shadcn Vue** - Beautiful UI components
- **Tailwind CSS** - Utility-first CSS
- **Pinia** - State management
- **Vue Router** - Routing
- **Axios** - HTTP client
- **Lucide Vue** - Icon library

## Setup Instructions

### 1. Install Dependencies

```bash
cd frontend
npm install
```

### 2. Install Shadcn Vue Components

The project uses Shadcn Vue components. To add more components as needed:

```bash
# Add sidebar component (already configured in layout)
npx shadcn-vue@latest add sidebar

# Add other components as needed
npx shadcn-vue@latest add button
npx shadcn-vue@latest add card
npx shadcn-vue@latest add table
npx shadcn-vue@latest add dialog
npx shadcn-vue@latest add badge
npx shadcn-vue@latest add dropdown-menu
```

### 3. Development Server

```bash
npm run dev
```

The app will be available at `http://localhost:3000`

### 4. Build for Production

```bash
npm run build
```

### 5. Preview Production Build

```bash
npm run preview
```

## Project Structure

```
frontend/
├── src/
│   ├── api/              # API client and services
│   │   └── client.ts     # Axios instance
│   ├── assets/           # CSS and static assets
│   │   └── index.css     # Global styles with Tailwind
│   ├── components/       # Reusable Vue components
│   │   └── ui/           # Shadcn UI components
│   ├── layouts/          # Layout components
│   │   └── DashboardLayout.vue  # Main dashboard layout
│   ├── lib/              # Utility functions
│   │   └── utils.ts      # Helper functions
│   ├── router/           # Vue Router configuration
│   │   └── index.ts      # Route definitions
│   ├── stores/           # Pinia stores
│   │   └── theme.ts      # Theme management
│   ├── types/            # TypeScript type definitions
│   │   └── index.ts      # Shared types
│   ├── views/            # Page components
│   │   ├── DashboardView.vue
│   │   ├── WorkflowsView.vue
│   │   ├── ExecutionsView.vue
│   │   └── AnalyticsView.vue
│   ├── App.vue           # Root component
│   └── main.ts           # Application entry point
├── public/               # Static assets
├── index.html            # HTML entry point
├── package.json          # Dependencies
├── tsconfig.json         # TypeScript config
├── vite.config.ts        # Vite config
└── tailwind.config.js    # Tailwind config
```

## Features

### ✅ Implemented

- Modern responsive layout with collapsible sidebar
- Dark/Light theme toggle
- Vue Router navigation
- TypeScript support
- Tailwind CSS styling
- Pinia state management
- API client setup with Axios
- Proxy configuration for backend API

### 🚧 To Be Implemented (As Requested)

- Workflows management (CRUD)
- Execution monitoring
- Analytics dashboard
- Real-time updates
- Data visualization

## API Configuration

The frontend is configured to proxy API requests to the backend:

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- Proxy: `/api` → `http://localhost:8080/api`

## Adding Shadcn Components

When you need a new component:

1. Check available components: https://www.shadcn-vue.com/docs/components
2. Install: `npx shadcn-vue@latest add [component-name]`
3. Import and use in your Vue components

Example:
```bash
npx shadcn-vue@latest add button
```

Then in your component:
```vue
<script setup lang="ts">
import { Button } from '@/components/ui/button'
</script>

<template>
  <Button>Click me</Button>
</template>
```

## Theme Customization

Modify colors and styles in:
- `tailwind.config.js` - Tailwind configuration
- `src/assets/index.css` - CSS variables for light/dark themes

## Next Steps

1. Install dependencies: `npm install`
2. Start dev server: `npm run dev`
3. Tell me which API integration you want to implement first:
   - Workflows list and creation
   - Execution monitoring
   - Analytics dashboard
   - Or something else

The layout is ready - let's build the features! 🚀
