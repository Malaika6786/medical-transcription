import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

// Professional Medical Theme - Dark Mode
// Inspired by modern healthcare dashboards with excellent contrast
const customDarkTheme = {
  dark: true,
  colors: {
    // Core backgrounds - deep navy with subtle blue undertones
    background: '#0D1117',
    surface: '#161B22',
    'surface-variant': '#21262D',
    'surface-bright': '#30363D',
    
    // Primary - Medical Teal (calming, professional, accessible)
    primary: '#00D9C4',
    'primary-darken-1': '#00B3A0',
    'primary-lighten-1': '#4DE8D9',
    
    // Secondary - Purple (for accents and interactive elements)
    secondary: '#8B5CF6',
    'secondary-darken-1': '#7C3AED',
    'secondary-lighten-1': '#A78BFA',
    
    // Semantic colors with good contrast
    error: '#F87171',
    info: '#60A5FA',
    success: '#4ADE80',
    warning: '#FBBF24',
    
    // Text colors - carefully chosen for readability
    'on-background': '#F0F6FC',
    'on-surface': '#E6EDF3',
    'on-surface-variant': '#8B949E',
    'on-primary': '#0D1117',
    'on-secondary': '#FFFFFF',
    'on-error': '#0D1117',
    'on-info': '#0D1117',
    'on-success': '#0D1117',
    'on-warning': '#0D1117',
  },
  variables: {
    'border-color': '#30363D',
    'border-opacity': 0.8,
    'high-emphasis-opacity': 1,
    'medium-emphasis-opacity': 0.7,
    'disabled-opacity': 0.38,
    'idle-opacity': 0.1,
    'hover-opacity': 0.08,
    'focus-opacity': 0.12,
    'selected-opacity': 0.16,
    'activated-opacity': 0.24,
    'pressed-opacity': 0.16,
    'dragged-opacity': 0.08,
    'theme-kbd': '#161B22',
    'theme-on-kbd': '#E6EDF3',
    'theme-code': '#161B22',
    'theme-on-code': '#E6EDF3',
  }
}

// Professional Medical Theme - Light Mode
// Clean, clinical aesthetic with excellent text contrast
const customLightTheme = {
  dark: false,
  colors: {
    // Core backgrounds - pure whites with subtle warmth
    background: '#FAFBFC',
    surface: '#FFFFFF',
    'surface-variant': '#F3F4F6',
    'surface-bright': '#E5E7EB',
    
    // Primary - Deeper teal for better contrast on light backgrounds
    primary: '#0D9488',
    'primary-darken-1': '#0F766E',
    'primary-lighten-1': '#14B8A6',
    
    // Secondary - Deeper purple for readability
    secondary: '#7C3AED',
    'secondary-darken-1': '#6D28D9',
    'secondary-lighten-1': '#8B5CF6',
    
    // Semantic colors - darker variants for light mode contrast
    error: '#DC2626',
    info: '#2563EB',
    success: '#16A34A',
    warning: '#D97706',
    
    // Text colors - high contrast for readability
    'on-background': '#111827',
    'on-surface': '#1F2937',
    'on-surface-variant': '#4B5563',
    'on-primary': '#FFFFFF',
    'on-secondary': '#FFFFFF',
    'on-error': '#FFFFFF',
    'on-info': '#FFFFFF',
    'on-success': '#FFFFFF',
    'on-warning': '#FFFFFF',
  },
  variables: {
    'border-color': '#D1D5DB',
    'border-opacity': 1,
    'high-emphasis-opacity': 1,
    'medium-emphasis-opacity': 0.7,
    'disabled-opacity': 0.38,
    'idle-opacity': 0.04,
    'hover-opacity': 0.08,
    'focus-opacity': 0.12,
    'selected-opacity': 0.12,
    'activated-opacity': 0.16,
    'pressed-opacity': 0.16,
    'dragged-opacity': 0.08,
    'theme-kbd': '#F3F4F6',
    'theme-on-kbd': '#1F2937',
    'theme-code': '#F3F4F6',
    'theme-on-code': '#1F2937',
  }
}

export default createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: 'customDarkTheme',
    themes: {
      customDarkTheme,
      customLightTheme,
    }
  },
  defaults: {
    VBtn: {
      rounded: 'lg',
      fontWeight: '600',
    },
    VCard: {
      rounded: 'xl',
      elevation: 0,
    },
    VTextField: {
      variant: 'outlined',
      density: 'comfortable',
      rounded: 'lg',
    },
    VSelect: {
      variant: 'outlined',
      density: 'comfortable',
      rounded: 'lg',
    },
    VChip: {
      rounded: 'lg',
    }
  }
})

