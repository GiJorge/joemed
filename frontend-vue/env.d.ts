/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// Add this so TypeScript stops throwing errors for CSS files
declare module '*.css' {
  const content: { [className: string]: string };
  export default content;
}