import { driver, DriveStep } from 'driver.js'
import 'driver.js/dist/driver.css'

// Tour definitions for different pages/features
export type TourType = 'welcome' | 'ambient' | 'file-transcription' | 'saved-sessions'

// Welcome/Home page tour
const welcomeTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Corti Medical Transcription',
      description: 'This application helps you transcribe clinical conversations and generate professional medical documents. Let\'s explore the main features.',
    }
  },
  {
    element: '.v-app-bar',
    popover: {
      title: 'Navigation Bar',
      description: 'Access all features from here. The menu adapts based on your user role and permissions.',
      side: 'bottom',
      align: 'center'
    }
  },
  {
    element: '[data-tour="nav-ambient"]',
    popover: {
      title: 'Ambient AI',
      description: 'Real-time transcription of clinical conversations with automatic document generation.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="nav-file-transcription"]',
    popover: {
      title: 'File Transcription',
      description: 'Upload pre-recorded audio files (WAV, MP3, etc.) for transcription with speaker identification.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="nav-saved-sessions"]',
    popover: {
      title: 'Saved Sessions',
      description: 'View and manage your saved transcription sessions. You can store up to 5 sessions.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="nav-demo-guide"]',
    popover: {
      title: 'Demo Guide',
      description: 'Comprehensive documentation and interactive tours for all features.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="theme-toggle"]',
    popover: {
      title: 'Theme Toggle',
      description: 'Switch between light and dark mode for your preferred viewing experience.',
      side: 'bottom',
      align: 'end'
    }
  },
  {
    popover: {
      title: 'Role-Based Access',
      description: 'Different user roles have access to different features. The navigation menu only shows features available to you.',
    }
  },
  {
    popover: {
      title: 'Ready to Start!',
      description: 'Click on any feature to begin. Each page also has its own guided tour that you can launch from the Demo Guide page.',
    }
  }
]

// Ambient Session page tour
const ambientTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Ambient AI',
      description: 'This module captures real-time clinical conversations and generates structured documentation. Let\'s walk through the workflow.',
    }
  },
  {
    element: '[data-tour="language-select"]',
    popover: {
      title: 'Step 1: Select Language',
      description: 'Choose the language being spoken for accurate transcription.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="microphone-select"]',
    popover: {
      title: 'Step 2: Select Microphone',
      description: 'Choose your audio input device. For best results, use a dedicated microphone in a quiet environment.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="start-recording"]',
    popover: {
      title: 'Step 3: Start Recording',
      description: 'Click this button to begin capturing audio. The transcription starts immediately.',
      side: 'right',
      align: 'start'
    }
  },
  {
    element: '[data-tour="live-transcript"]',
    popover: {
      title: 'Step 4: Live Transcript',
      description: 'During recording, your conversation appears here in real-time. Click "Stop Recording" when finished.',
      side: 'left',
      align: 'start'
    }
  },
  {
    popover: {
      title: 'Step 5: Generate Document',
      description: 'After stopping the recording, the "Generate Document" section becomes active. Select a template (SOAP Note, GP Letter, etc.) and click "Generate" to create structured clinical documentation.',
    }
  },
  {
    popover: {
      title: 'Step 6: Edit & Save',
      description: 'The generated document can be edited using the rich text editor. Add formatting, make corrections, then click "Save Session" to store it for future reference.',
    }
  },
  {
    popover: {
      title: 'Ready to Try!',
      description: 'That\'s the complete workflow! Start by selecting your language and microphone, then click "Start Recording" to begin.',
    }
  }
]

// File Transcription page tour
const fileTranscriptionTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to File Transcription',
      description: 'Upload pre-recorded audio files for transcription. Let\'s walk through the complete workflow.',
    }
  },
  {
    element: '[data-tour="file-upload"]',
    popover: {
      title: 'Step 1: Upload Audio File',
      description: 'Drag and drop or click to select an audio file. Supports WAV, MP3, M4A, FLAC, OGG, and WEBM formats.',
      side: 'bottom',
      align: 'center'
    }
  },
  {
    element: '[data-tour="upload-language"]',
    popover: {
      title: 'Step 2: Select Language',
      description: 'Choose the language spoken in the audio file for accurate transcription.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="upload-button"]',
    popover: {
      title: 'Step 3: Start Transcription',
      description: 'Click this button to upload and begin transcribing your audio file. A progress indicator will show the status.',
      side: 'top',
      align: 'center'
    }
  },
  {
    popover: {
      title: 'Step 4: Review Transcript',
      description: 'Once processing completes, the transcript appears on the right side with timestamps and speaker identification.',
    }
  },
  {
    popover: {
      title: 'Step 5: Generate Document',
      description: 'After transcription, select a template from the dropdown and click "Generate Document" to create structured clinical notes.',
    }
  },
  {
    popover: {
      title: 'Step 6: Edit & Export',
      description: 'Edit the generated document with the rich text editor. You can copy, download, or save the session for future reference.',
    }
  },
  {
    popover: {
      title: 'Ready to Try!',
      description: 'That\'s the complete workflow! Start by uploading an audio file using the upload area above.',
    }
  }
]

// Saved Sessions page tour
const savedSessionsTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Saved Sessions',
      description: 'This page shows all your saved transcription sessions. You can store up to 5 sessions per user.',
    }
  },
  {
    element: '[data-tour="session-filters"]',
    popover: {
      title: 'Filter by Type',
      description: 'Use these tabs to filter sessions by type: All, Ambient, File Transcription, or Dictation.',
      side: 'bottom',
      align: 'start'
    }
  },
  {
    element: '[data-tour="session-card"]',
    popover: {
      title: 'Session Cards',
      description: 'Each card shows a saved session. Click to view details, edit the document, copy content, or download.',
      side: 'right',
      align: 'start'
    }
  },
  {
    popover: {
      title: 'Viewing a Session',
      description: 'When you open a session, you can switch between the Transcript and Document tabs. Use Edit mode to make changes to the document.',
    }
  },
  {
    popover: {
      title: 'Storage Limit',
      description: 'You can save up to 5 sessions. When the limit is reached, saving a new session automatically removes the oldest one.',
    }
  },
  {
    popover: {
      title: 'Getting Started',
      description: 'To save a session, complete a transcription in Ambient AI or File Transcription, generate a document, then click "Save Session".',
    }
  }
]

// Get tour steps by type
const getTourSteps = (tourType: TourType): DriveStep[] => {
  switch (tourType) {
    case 'welcome':
      return welcomeTourSteps
    case 'ambient':
      return ambientTourSteps
    case 'file-transcription':
      return fileTranscriptionTourSteps
    case 'saved-sessions':
      return savedSessionsTourSteps
    default:
      return welcomeTourSteps
  }
}

// Check if tour has been completed
const isTourCompleted = (tourType: TourType): boolean => {
  const completed = localStorage.getItem(`tour_completed_${tourType}`)
  return completed === 'true'
}

// Mark tour as completed
const markTourCompleted = (tourType: TourType): void => {
  localStorage.setItem(`tour_completed_${tourType}`, 'true')
}

// Reset tour completion status
export const resetTour = (tourType: TourType): void => {
  localStorage.removeItem(`tour_completed_${tourType}`)
}

// Reset all tours
export const resetAllTours = (): void => {
  const tourTypes: TourType[] = ['welcome', 'ambient', 'file-transcription', 'saved-sessions']
  tourTypes.forEach(type => resetTour(type))
}

// Start a product tour
export const startTour = (tourType: TourType, forceStart = false): void => {
  // Check if tour already completed (unless forced)
  if (!forceStart && isTourCompleted(tourType)) {
    return
  }

  const steps = getTourSteps(tourType)
  
  // Filter out steps with elements that don't exist
  const filteredSteps = steps.filter(step => {
    if (!step.element) return true // Keep steps without elements (modal popups)
    const el = document.querySelector(step.element as string)
    return el !== null
  })

  if (filteredSteps.length === 0) return

  const driverObj = driver({
    showProgress: true,
    showButtons: ['next', 'previous', 'close'],
    steps: filteredSteps,
    nextBtnText: 'Next →',
    prevBtnText: '← Back',
    doneBtnText: 'Done ✓',
    progressText: '{{current}} of {{total}}',
    onDestroyed: () => {
      markTourCompleted(tourType)
    },
    popoverClass: 'xstek-tour-popover'
  })

  // Small delay to ensure DOM is ready
  setTimeout(() => {
    driverObj.drive()
  }, 500)
}

// Composable function
export function useProductTour() {
  return {
    startTour,
    resetTour,
    resetAllTours,
    isTourCompleted
  }
}
