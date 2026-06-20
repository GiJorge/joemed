import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.jorge.medi',
  appName: 'MindfulMoments',
  webDir: 'dist',
  // Add this block below to fix the mixed content HTTPS block:
  server: {
    androidScheme: 'http'
  }
};

export default config;
