<script setup>
import { ref, onMounted, computed } from 'vue'

import { offlineStorage } from '../offlineStorage' // Import our storage helper
import axios from 'axios' // Import axios for progress tracking
import { Preferences } from '@capacitor/preferences';

const serverIP = ref('');
const ipInputField = ref('');
const isIPConfigured = ref(false);

const API_URL = ref('');

const showPasswordModal = ref(false);
const adminPasswordInput = ref('');
// Add this near your other ref properties
const userRole = ref('guest'); // Default to 'guest' for a clean meditator experience!

function toggleRole() {
  if (userRole.value === 'admin') {
    // Switching back down to guest requires no authentication check
    userRole.value = 'guest';
    console.log("Logged out of admin panel access.");
  } else {
    // Open password interface prompt
    adminPasswordInput.value = '';
    showPasswordModal.value = true;
  }
}

const selectedFileName = ref('');

const isDarkMode = ref(true); // Default to dark mode for meditation vibes!

function toggleTheme() {
  isDarkMode.value = !isDarkMode.value;
}

//const API_URL = "http://192.168.8.30:8080"
const downloadedTracks = ref(new Set()) // Tracks stored locally
const downloadingId = ref(null)        // Tracks currently downloading

// Modal UI state management
const showDeleteModal = ref(false);
const trackToDelete = ref(null);

const tracks = ref([])
const currentTrack = ref(null)
const isPlaying = ref(false)

// Time tracking states
const currentTime = ref(0) // in seconds
const duration = ref(0)    // in seconds

let audioPlayer = null

// --- Upload & Modal States ---
const newTrack = ref({ title: '', duration: '' })
let selectedFile = null
const uploadProgress = ref(0)       // Percentage (0 to 100)
const isUploading = ref(false)      // Toggles progress bar visibility
const showModal = ref(false)        // Toggles modal visibility
const modalType = ref('confirm')
const modalTitle = ref('')
const modalMessage = ref('')

// --- Place these near your other ref variables ---
const editingTrackId = ref(null);
const editForm = ref({ title: '', duration: '' });

// Start inline editing mode for a specific track
function startEdit(track) {
  editingTrackId.value = track.id;
  editForm.value = { title: track.title, duration: track.duration };
}

// Cancel editing mode
function cancelEdit() {
  editingTrackId.value = null;
}

// Save the edited changes to the Go Server
async function saveEdit(trackId) {
  if (!editForm.value.title.trim() || !editForm.value.duration.trim()) {
    alert("Fields cannot be empty.");
    return;
  }


   
  try {
    const response = await fetch(`${API_URL.value}/api/tracks/${trackId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editForm.value)
    });

    if (response.ok) {
      editingTrackId.value = null; // Exit edit mode
      await loadTracks(); // Refresh the UI with updated metadata 🎉
    } else {
      alert("Failed to update the track on the server.");
    }
  } catch (error) {
    console.error("Error updating track:", error);
    alert("Connection error trying to reach server.");
  }
}


async function verifyAdminPassword() {
  try {
    // 🌟 FIXED: Removed ${API_URL} so it hits the relative path /api/verify-admin directly
    const response = await fetch('/api/verify-admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: adminPasswordInput.value })
    });

    if (response.ok) {
      userRole.value = 'admin';
      showPasswordModal.value = false;
      console.log("Admin authorization accepted! Welcome back.");
    } else {
      alert("Incorrect admin passcode. Access denied.");
    }
  } catch (error) {
    console.error("Authentication server down:", error);
    alert("Could not reach authentication server.");
  }
}





// const handleFileSelect = (event) => {
//   selectedFile = event.target.files[0]
// }



const handleFileSelect = (event) => {
  const file = event.target.files[0];
  if (file) {
    // 1. Save the file object so your upload function can read it 🎉
    selectedFile = file; 
    
    // 2. Capture the filename string for your custom label text
    selectedFileName.value = file.name; 
    
    console.log("File selected and ready:", file.name);
  }
};

// Opens the modal custom layout
const triggerModal = (type, title, message) => {
  modalType.value = type
  modalTitle.value = title
  modalMessage.value = message
  showModal.value = true
}


// Step 1: Trigger confirmation modal first
const confirmUpload = () => {
  if (!selectedFile) return
  triggerModal(
    'confirm', 
    'Confirm Upload', 
    `Are you sure you want to upload "${newTrack.value.title}" to the server?`
  )
}



function confirmBackToSessions() {
  // If no track is playing, or it's paused/finished, let them pass instantly
  if (!currentTrack.value || !isPlaying.value) {
    exitCurrentTrack();
    return;
  }

  // Show a gentle confirmation dialog
  const leaveSession = confirm("Are you sure you want to leave your active meditation session?");
  if (leaveSession) {
    exitCurrentTrack();
  }
}

// Helper logic to cleanly clear out the active streaming states
function exitCurrentTrack() {
  if (audioPlayer) {
    audioPlayer.pause();
  }
  isPlaying.value = false;
  currentTrack.value = null;
}




// Step 2: If confirmed, execute the real upload using Axios
const executeUpload = async () => {
  showModal.value = false // close confirmation modal
  isUploading.value = true
  uploadProgress.value = 0

  const formData = new FormData()
  formData.append('title', newTrack.value.title)
  formData.append('duration', newTrack.value.duration)
  formData.append('audioFile', selectedFile)

  try {
    const response = await axios.post(`${API_URL.value}/api/upload`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      // Axios native hook to track progress percentages
      onUploadProgress: (progressEvent) => {
        const percentCompleted = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        uploadProgress.value = percentCompleted
      }
    })

    if (response.status === 201) {
      // Trigger notification modal upon completion
      triggerModal('notify', 'Success! 🎉', 'Your meditation track has been securely saved to the server.')
      newTrack.value = { title: '', duration: '' }
      selectedFile = null
      loadTracks() // Refresh the track list
    }
  } catch (error) {
    console.error("Upload error:", error)
    triggerModal('notify', 'Upload Failed ❌', 'Something went wrong while transferring the audio track.')
  } finally {
    isUploading.value = false
  }
}


const loadTracks = async () => {
  try {
    // 1. Try to get fresh data from the dynamic Go server target
    const response = await fetch(`${API_URL.value}/api/tracks`);
    if (!response.ok) throw new Error("Failed to pull server tracks");
    
    const data = await response.json();
    tracks.value = data;
    
    // Back up the track metadata structure locally for offline fallback reference
    localStorage.setItem('cached_track_list', JSON.stringify(data));
    
    // Check local storage file binary sync states
    for (const track of data) {
      const offlineBlob = await offlineStorage.getTrack(track.id);
      if (offlineBlob) {
        downloadedTracks.value.add(track.id);
      }
    }
  } catch (error) {
    console.warn("🌐 Server unreachable. Attempting fallback to local offline metadata...", error);
    
    // 2. FALLBACK: Load the track structure from localStorage if server is offline
    const localSavedList = localStorage.getItem('cached_track_list');
    
    if (localSavedList) {
      tracks.value = JSON.parse(localSavedList);
      
      // Filter the list to ONLY show tracks that are actually downloaded on disk
      const availableTracks = [];
      for (const track of tracks.value) {
        const offlineBlob = await offlineStorage.getTrack(track.id);
        if (offlineBlob) {
          downloadedTracks.value.add(track.id);
          availableTracks.push(track);
        }
      }
      
      // Show the user only what they can actually play right now
      tracks.value = availableTracks;
      
      // Notify them gently that they are browsing downloaded tracks
      alert("Operating in Offline Mode. Showing saved local tracks.");
    } else {
      // If there's no server and absolutely no local cache, send them back to the network screen
      alert("Could not connect to the meditation server, and no offline cache was found. Please check your IP!");
      isIPConfigured.value = false; 
    }
  }
};




// Download file from Go backend and store into IndexedDB
const downloadForOffline = async (track) => {
  downloadingId.value = track.id
  try {
    const response = await fetch(track.url)
    const blob = await response.blob()
    
    await offlineStorage.saveTrack(track.id, blob)
    downloadedTracks.value.add(track.id)
    
    // FIX: Trigger your gorgeous new notification modal instead of a boring alert!
    triggerModal(
      'notify', 
      'Saved Offline! 💾', 
      `"${track.title}" has been successfully saved to your device. You can now play this track anytime without internet.`
    )

  } catch (error) {
    console.error("Failed downloading track:", error)
    triggerModal('notify', 'Download Failed ❌', 'Could not save track offline. Check your server connection.')
  } finally {
    downloadingId.value = null
  }
}



// Fetch tracks from Go Backend
onMounted(async () => {

// Check if an IP address was saved during a previous app launch
  const { value } = await Preferences.get({ key: 'meditation_server_ip' });
  
  if (value) {
    serverIP.value = value;
    API_URL.value = `http://${value}`;
    isIPConfigured.value = true;
    loadTracks(); // Safely fire your core track loading logic
  } else {
    isIPConfigured.value = false; // Trigger the setup screen layout
  }

 //loadTracks()

})

async function saveServerIP() {
  let cleanIP = ipInputField.value.trim();
  
  // Strip out http:// or trailing slashes if you accidentally typed them
  cleanIP = cleanIP.replace(/^https?:\/\//, '').replace(/\/$/, '');

  if (!cleanIP) {
    alert("Please enter a valid IP address or server URL!");
    return;
  }

  // Save the address permanently to the local storage cluster
  await Preferences.set({
    key: 'meditation_server_ip',
    value: cleanIP
  });

  serverIP.value = cleanIP;
  API_URL.value = `http://${cleanIP}`;
  isIPConfigured.value = true;
  
  loadTracks();
}


// Optional helper button logic to allow resetting the IP configuration layout later
async function resetServerIP() {
  if(confirm("Change server IP destination target?")) {
    await Preferences.remove({ key: 'meditation_server_ip' });
    isIPConfigured.value = false;
  }
}





// 🗑️ 2. Open confirmation modal dialog
function confirmDelete(track) {
  trackToDelete.value = track;
  showDeleteModal.value = true;
}

async function executeDelete() {
  // Hide the modal immediately
  showDeleteModal.value = false;
  
  if (!trackToDelete.value) return;

  try {
  const response = await fetch(`${API_URL.value}/api/tracks/${trackToDelete.value.id}`, {
      method: 'DELETE',
      headers: {
        'Accept': 'application/json'
      }
    });
 

    // If the server returns a successful 200 OK status code, it was deleted!
    if (response.ok) {
      console.log("Track deleted successfully from server.");
      trackToDelete.value = null;
      
      // 🎉 Automatically call your function to reload the track list layout instantly
      await loadTracks(); 
    } else {
      const errorText = await response.text();
      console.error("Server rejected delete request:", errorText);
      alert("Failed to delete track. Server returned an error.");
    }
  } catch (error) {
    console.error("Actual network error:", error);
    alert("Connection error trying to reach server.");
  }
}


const uploadTrack = async () => {
  if (!selectedFile) return

  const formData = new FormData()
  formData.append('title', newTrack.value.title)
  formData.append('duration', newTrack.value.duration)
  formData.append('audioFile', selectedFile)

  try {
    const res = await fetch(`${API_URL.value}/api/upload`, {
      method: 'POST',
      body: formData
    })
    
    if (res.ok) {
      alert("Track uploaded successfully!")
      newTrack.value = { title: '', duration: '' } // reset form
      loadTracks() // reload dynamic track list from SQLite
    } else {
      alert("Upload failed.")
    }
  } catch (error) {
    console.error("Error uploading track:", error)
  }
}


// Control Audio Playback
const playTrack = async (track) => {




currentTrack.value = track;
  
  // Create or update audio element targeting the new dynamic IP address
  if (audioPlayer) {
    audioPlayer.pause();
  }
  
  // 🌟 CHANGED: Pointing the audio stream source straight to the user-entered IP
  const streamUrl = `${API_URL.value}/audio/${track.filename}`;
  audioPlayer = new Audio(streamUrl);
  
  audioPlayer.play();
  isPlaying.value = true;




  

  currentTrack.value = track
  currentTime.value = 0
  duration.value = 0
  isPlaying.value = true

  // FIX: Create the bell instance immediately inside the user click event
  const startBell = new Audio('/bell1.mp3')
  startBell.play().catch(err => console.log("Start bell blocked:", err))

  // Determine Source URL: Server Stream vs Local Storage Blob
  let audioSourceUrl = track.url
  const offlineBlob = await offlineStorage.getTrack(track.id)
  
  if (offlineBlob) {
    console.log("Playing from LOCAL OFFLINE CACHE (IndexedDB) 🎉")
    audioSourceUrl = URL.createObjectURL(offlineBlob)
  } else {
    console.log("Streaming ONLINE from Go Server 🌐")
  }

  //  FIXED: Changed track.url to audioSourceUrl so it uses the local blob!
  audioPlayer = new Audio(audioSourceUrl)
  
  // Optional: Delay the music slightly so the bell can ring solo
  setTimeout(() => {
    if (isPlaying.value) {
      audioPlayer.play().catch(err => console.log("Music track blocked:", err))
    }
  }, 2000)

  audioPlayer.onloadedmetadata = () => {
    duration.value = audioPlayer.duration
  }

  audioPlayer.ontimeupdate = () => {
    currentTime.value = audioPlayer.currentTime
  }

  audioPlayer.onended = () => {
    isPlaying.value = false
    currentTime.value = duration.value
    
    // FIX: Create a fresh bell instance for the end chime
    const endBell = new Audio('/bell1.mp3')
    endBell.play().catch(err => console.log("End bell blocked:", err))
  }
}





const togglePlay = () => {
  if (!audioPlayer) return
  if (isPlaying.value) {
    audioPlayer.pause()
  } else {
    audioPlayer.play()
  }
  isPlaying.value = !isPlaying.value
}


const playBellChime = () => {
  const bell = new Audio('/bell1.mp3') // Ensure this file is in your public folder
  bell.volume = 0.8 // You can control the bell volume separately from the music!
  bell.play().catch(err => console.log("Audio playback interrupted:", err))
}

// --- Helper Functions for Formatting & Math ---

// Formats seconds into MM:SS format
const formatTime = (timeInSeconds) => {
  if (isNaN(timeInSeconds)) return "00:00"
  const minutes = Math.floor(timeInSeconds / 60)
  const seconds = Math.floor(timeInSeconds % 60)
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}

const formattedCurrentTime = computed(() => formatTime(currentTime.value))
const formattedRemainingTime = computed(() => formatTime(duration.value - currentTime.value))

// SVG Animation Circle Math
const radius = 100
const circumference = 2 * Math.PI * radius
const strokeDashoffset = computed(() => {
  if (!duration.value) return circumference
  const progress = currentTime.value / duration.value
  return circumference - (progress * circumference)
})
</script>
<template>

<div v-if="!isIPConfigured" class="setup-screen-wrapper">
    <div class="setup-card">
      <span class="setup-icon">🌐</span>
      <h2>Connect to Server</h2>
      <p>Enter your local computer network IP address to discover meditation assets.</p>
      
      <form @submit.prevent="saveServerIP">
        <input 
          v-model="ipInputField" 
          type="text" 
          placeholder="e.g., 192.168.8.30" 
          class="setup-input"
          required 
        />
        <button type="submit" class="setup-submit-btn">Connect Interface</button>
      </form>
    </div>
  </div>

  <div :class="['meditation-player', isDarkMode ? 'dark-theme' : 'light-theme']">
    <header class="app-header"><h1 class="cen">🧘</h1>
      <h1>Wisdom Ethiopia Meditation </h1>
      
 
      <div class="header-actions">
        <button @click="toggleRole" class="btn-icon">
          🔑 {{ userRole === 'admin' ? 'Guest' : ' Admin' }}
        </button>
        <button @click="resetServerIP" class="btn-icon" title="Reset Server Target">⚙️ IP</button>
        <button @click="toggleTheme" class="btn-icon">
          {{ isDarkMode ? '☀️ Light' : '🌙 Dark' }}
        </button>
        <button @click="loadTracks()" class="btn-icon" title="Refresh Tracks">
          🔄 Refresh
        </button>
      </div> 
      <p class="cen">Your daily peace space</p>
    </header>

  <div class="meditation-container cen">
    <h2 v-if="!currentTrack">Choose Your Session Length</h2>
    
    <div v-if="!currentTrack && userRole === 'admin'" class="upload-section">
      <h3>➕ Upload New Track</h3>
      <form @submit.prevent="confirmUpload" class="upload-form">
        <input v-model="newTrack.title" type="text" placeholder="Track Title (e.g. Zen Focus)" required />
        <input v-model="newTrack.duration" type="text" placeholder="Duration (e.g. 25m)" required />
        
        <div class="file-upload-container">
          <input type="file" id="native-file-input" @change="handleFileSelect" accept="audio/mp3,audio/mpeg" required class="visually-hidden" />
          <label for="native-file-input" class="custom-file-label">
            🎵 {{ selectedFileName || 'Choose Audio File' }}
          </label>
        </div>

        

        <button type="submit" class="upload-btn" :disabled="isUploading">Upload to Server</button>
      </form>

      <div v-if="isUploading" class="progress-container">
        <div class="progress-bar" :style="{ width: uploadProgress + '%' }"></div>
        <span class="progress-text">Uploading: {{ uploadProgress }}%</span>
      </div>
    </div>

     <div v-if="showModal" class="modal-overlay">
      <div class="modal-card">
        <h4>{{ modalTitle }}</h4>
        <p>{{ modalMessage }}</p>

        <div class="modal-actions">
          <template v-if="modalType === 'confirm'">
            <button @click="executeUpload" class="btn-confirm">Yes, Upload</button>
            <button @click="showModal = false" class="btn-cancel">Cancel</button>
          </template>
          <template v-else>
            <button @click="showModal = false" class="btn-close">Dismiss</button>
          </template>
        </div>
      </div>
    </div>

    <div v-if="!currentTrack" class="track-list">
     <div v-for="track in tracks" :key="track.id" class="track-row">
  
  <template v-if="editingTrackId === track.id && userRole === 'admin'">
    <div class="edit-inline-form">
      <input v-model="editForm.title" type="text" class="edit-input" />
      <input v-model="editForm.duration" type="text" class="edit-input spec-width" />
      <div class="track-actions-wrapper">
        <button @click="saveEdit(track.id)" class="btn-action-save">💾 Save</button>
        <button @click="cancelEdit" class="btn-action-cancel">❌</button>
      </div>
    </div>
  </template>

  <template v-else>
    <button class="track-select-btn" @click="playTrack(track)">
      <span class="track-title-text">{{ track.title }} ({{ track.duration }})</span>
      
      <div class="track-waveform">
        <span class="bar bar-1"></span>
        <span class="bar bar-2"></span>
        <span class="bar bar-3"></span>
        <span class="bar bar-4"></span>
        <span class="bar bar-5"></span>
        <span class="bar bar-6"></span>
        <span class="bar bar-7"></span>
      </div>

      <span v-if="downloadedTracks.has(track.id)" class="badge-offline">💾 Saved</span>
    </button>

    <div class="track-actions-wrapper">
      <button v-if="userRole === 'admin'" @click="startEdit(track)" class="btn-edit-icon" title="Edit">✏️</button>
      <button v-if="userRole === 'admin'" @click="confirmDelete(track)" class="btn-delete" title="Delete">🗑️</button>
      
      <button 
        v-if="!downloadedTracks.has(track.id)" 
        class="download-icon-btn"
        @click.stop="downloadForOffline(track)"
        :disabled="downloadingId === track.id"
      >
        {{ downloadingId === track.id ? '⏳' : '⬇️' }}
      </button>
    </div>
  </template>

</div>
    </div>

       <div v-else class="player-screen  ">
   <button class="back-btn" @click="confirmBackToSessions">
  ← Back to Sessions
</button>

      <h3>{{ currentTrack.title }}</h3>

      <div class="timer-visual">
        <svg width="240" height="240" viewBox="0 0 240 240">
          <circle cx="120" cy="120" :r="radius" class="circle-bg" />
          <circle
            cx="120" cy="120" :r="radius"
            class="circle-progress"
            :style="{ strokeDasharray: circumference, strokeDashoffset: strokeDashoffset }"
          />
        </svg>

        <div class="time-overlay">
          <span class="elapsed-time">{{ formattedCurrentTime }}</span>
          <span class="remaining-time">-{{ formattedRemainingTime }}</span>
        </div>
      </div>

      <div class="controls">
        <button class="play-pause-btn" @click="togglePlay">
          {{ isPlaying ? '⏸' : '▶' }}
        </button>
      </div>
    </div>
    
    
<div v-if="showDeleteModal" class="modal-overlay">
      <div class="modal-card">
        <h3>Delete Track?</h3>
        <p>Are you sure you want to permanently delete <strong>{{ trackToDelete?.title }}</strong> from the server? This action cannot be undone.</p>

        <div class="modal-actions">
          <button @click="showDeleteModal = false" class="btn-cancel">Cancel</button>
          <button @click="executeDelete" class="btn-confirm-danger">Delete</button>
        </div>




        
      </div>
    </div>


  </div>
  </div>


  <div v-if="showPasswordModal" class="modal-overlay">
  <div class="modal-card">
    <h3>Enter Admin Password</h3>
    <p>Please enter your passcode to access upload and edit panels.</p>
    
    <form @submit.prevent="verifyAdminPassword">
      <input 
        v-model="adminPasswordInput" 
        type="password" 
        placeholder="••••••••" 
        class="edit-input" 
        style="text-align: center; margin-bottom: 20px;"
        required 
        ref="passwordInputField"
      />
      
      <div class="modal-actions">
        <button type="button" @click="showPasswordModal = false" class="btn-cancel">Cancel</button>
        <button type="submit" class="btn-confirm">Unlock</button>
      </div>
    </form>
  </div>
</div>
</template>

<style scoped>
/* ==========================================================================
   1. GLOBAL LAYOUT & CONTAINER (Fixed Margins for DeX)
   ========================================================================== */
.meditation-player {
  min-height: 100vh;
  padding: 24px;
  transition: background-color 0.3s ease, color 0.3s ease;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  box-sizing: border-box;
}

.meditation-container {
  width: 100%;
  margin: 0;
  padding-bottom: 60px;
}

h2 {
  font-size: 1.4rem;
  margin: 0 0 20px 0;
  font-weight: 600;
  letter-spacing: -0.5px;
}

/* ==========================================================================
   2. APP HEADER STYLING
   ========================================================================== */
.app-header {
  width: 100%;
  margin: 0 0 30px 0;
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
}

.app-header h1 {
  font-size: 1.6rem;
  margin: 0;
  grid-column: 1 / span 2;
  padding-bottom: 6px;

    text-align: center;

}
.cen{
  
    text-align: center;

}

.app-header p {
  grid-column: 1 / span 2;
  margin: 6px 0 0 0;
  font-size: 0.95rem;
  opacity: 0.7;
}

.header-actions {
  display: flex;
  gap: 10px;
  grid-column: 2;
}

.btn-icon {
  background: transparent;
  border: 1px solid currentColor;
  padding: 8px 14px;
  border-radius: 20px;
  cursor: pointer;
  font-size: 0.85rem;
  color: inherit;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
}

/* ==========================================================================
   3. MOBILE LAYOUT (1 Column Stacked)
   ========================================================================== */
.upload-section {
  padding: 20px;
  border-radius: 16px;
  margin-bottom: 24px;
  box-sizing: border-box;
}

.upload-section h3 {
  margin: 0 0 16px 0;
  font-size: 1.1rem;
}

.upload-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-form input[type="text"] {
  width: 100%;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid;
  font-size: 0.95rem;
  background: transparent;
  color: inherit;
  box-sizing: border-box;
}

.upload-btn {
  width: 100%;
  padding: 12px;
  border-radius: 8px;
  border: none;
  font-weight: 600;
  font-size: 0.95rem;
  cursor: pointer;
}

.track-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.track-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px;
  border-radius: 12px;
}

.track-select-btn {
  flex: 1;
  background: transparent;
  border: none;
  text-align: left;
  padding: 14px 16px;
  font-size: 1.05rem;
  font-weight: 500;
  color: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* ==========================================================================
   4. 🖥️ TABLET & SAMSUNG DEX RESPONSIVE LAYOUT (Grid Transformation)
   ========================================================================== */
@media (min-width: 768px) {

  
  .meditation-container {
    display: grid;
    /* Left column for upload, right column adapts to screen width */
    grid-template-columns: 320px 1fr;
    column-gap: 24px;
    row-gap: 20px;
    align-items: start;
  }
  
  h2 {
    grid-column: 1 / span 2;
  }

  .upload-section {
    grid-column: 1;
    
    position: sticky;
    top: 24px;
    margin-bottom: 0;
  }

  /* Turns the track list row into an auto-adjusting grid card layout */
  .track-list {
    grid-column: 2;
    display: grid;
    /* Automatically creates as many columns as will fit without squishing cards */
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }
  
  /* Formats rows into clean grid block cards */
  .track-row {
    flex-direction: column;
    align-items: stretch;
    padding: 16px;
    text-align: center;
  }

  .track-select-btn {
    text-align: center;
    flex-direction: column;
    gap: 10px;
    padding: 10px 0;
  }

  .track-actions-wrapper {
    display: flex;
    justify-content: center;
    gap: 12px;
    margin-top: 10px;
  }
}

/* ==========================================================================
   5. PROGRESS, MODALS & BUTTONS HELPERS
   ========================================================================== */
.badge-offline {
  font-size: 0.75rem;
  padding: 4px 8px;
  border-radius: 12px;
  font-weight: 600;
}

.btn-delete, .download-icon-btn {
  background: transparent;
  border: none;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.2rem;
  cursor: pointer;
  border-radius: 50%;
}

.progress-container {
  margin-top: 14px;
  background: rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  height: 20px;
  position: relative;
  overflow: hidden;
}
.progress-bar { background: #48bb78; height: 100%; }
.progress-text { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); font-size: 0.75rem; color: #fff; }

/* ==========================================================================
   6. AUDIO PLAYER SCREEN
   ========================================================================== */
.player-screen {
  grid-column: 1 / span 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 40px 0;
}
.back-btn { background: transparent; border: none; color: inherit; font-size: 1.2rem; cursor: pointer; margin-bottom: 24px;  }
.player-screen h3 { font-size: 1.8rem; margin-bottom: 30px; }
.timer-visual { position: relative; width: 240px; height: 240px; margin-bottom: 40px; }
.circle-bg { fill: none; stroke-width: 8px; }
.circle-progress { fill: none; stroke-width: 8px; stroke-linecap: round; transform: rotate(-90deg); transform-origin: 50% 50%; }
.time-overlay { position: absolute; top: 0; left: 0; width: 100%; height: 100%; display: flex; flex-direction: column; justify-content: center; align-items: center; }
.elapsed-time { font-size: 2.5rem; font-weight: 700; }
.remaining-time { font-size: 0.95rem; opacity: 0.6; margin-top: 4px; }
.play-pause-btn { width: 76px; height: 76px; border-radius: 50%; border: none; font-size: 2rem; display: flex; align-items: center; justify-content: center; cursor: pointer; }

/* ==========================================================================
   7. MODAL OVERLAY INTERFACES
   ========================================================================== */
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0, 0, 0, 0.55); backdrop-filter: blur(4px); display: flex; justify-content: center; align-items: center; z-index: 9999; padding: 20px; }
.modal-card { width: 100%; max-width: 400px; padding: 24px; border-radius: 20px; text-align: center; }
.modal-actions { display: flex; gap: 12px; justify-content: center; }
.modal-actions button { flex: 1; padding: 12px; border-radius: 8px; border: none; font-weight: 600; cursor: pointer; }
.btn-confirm { background: #48bb78; color: white; }
.btn-confirm-danger { background: #e53e3e; color: white; }
.btn-cancel { background: rgba(0,0,0,0.15); color: inherit; }
.btn-close { background: #4a5568; color: white; }

/* ==========================================================================
   8. SYSTEM THEMES (COLORS)
   ========================================================================== */
.dark-theme { background-color: #121214; color: #f7fafc; }
.dark-theme .upload-section { background-color: #1a1a1e; border: 1px solid #2d3748; }
.dark-theme .upload-form input[type="text"] { border-color: #4a5568; }
.dark-theme .upload-btn { background-color: #f7fafc; color: #121214; }
.dark-theme .track-row { background-color: #1a1a1e; border: 1px solid #2d3748; }
.dark-theme .badge-offline { background-color: #2d3748; color: #63b3ed; }
.dark-theme .circle-bg { stroke: #2d3748; }
.dark-theme .circle-progress { stroke: #63b3ed; }
.dark-theme .play-pause-btn { background-color: #f7fafc; color: #121214; }
.dark-theme .modal-card { background-color: #1a1a1e; border: 1px solid #2d3748; }

.light-theme { background-color: #f7fafc; color: #2d3748; }
.light-theme .upload-section { background-color: #ffffff; border: 1px solid #e2e8f0; }
.light-theme .upload-form input[type="text"] { border-color: #cbd5e0; }
.light-theme .upload-btn { background-color: #2d3748; color: #ffffff; }
.light-theme .track-row { background-color: #ffffff; border: 1px solid #e2e8f0; }
.light-theme .badge-offline { background-color: #ebf8ff; color: #2b6cb0; }
.light-theme .circle-bg { stroke: #e2e8f0; }
.light-theme .circle-progress { stroke: #3182ce; }
.light-theme .play-pause-btn { background-color: #2d3748; color: #ffffff; }
.light-theme .modal-card { background-color: #ffffff; }


.edit-inline-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  padding: 8px;
}
.edit-input {
  width: 100%;
  padding: 10px;
  border-radius: 6px;
  border: 1px solid currentColor;
  background: transparent;
  color: inherit;
  box-sizing: border-box;
}
.btn-action-save { background: #3182ce; color: white; border: none; padding: 8px 16px; border-radius: 6px; cursor: pointer; font-weight: 600; }
.btn-action-cancel { background: rgba(0,0,0,0.1); color: inherit; border: none; padding: 8px 12px; border-radius: 6px; cursor: pointer; }
.btn-edit-icon { background: transparent; border: none; width: 44px; height: 44px; display: flex; align-items: center; justify-content: center; font-size: 1.1rem; cursor: pointer; border-radius: 50%; }

/* Layout adjustments inside tablet/desktop viewports */
@media (min-width: 768px) {
  .edit-inline-form {
    align-items: center;
    text-align: center;
  }
}



/* Hides the ugly browser default button completely */
.visually-hidden {
  position: absolute !important;
  width: 1px !important;
  height: 1px !important;
  padding: 0 !important;
  margin: -1px !important;
  overflow: hidden !important;
  clip: rect(0, 0, 0, 0) !important;
  white-space: nowrap !important;
  border: 0 !important;
}

/* Styles our custom label beautifully */
.custom-file-label {
  display: block;
  width: 100%;
  padding: 14px;
  border: 2px dashed currentColor;
  border-radius: 8px;
  text-align: center;
  font-weight: 500;
  cursor: pointer;
  box-sizing: border-box;
  opacity: 0.8;
  transition: opacity 0.2s, background-color 0.2s;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.custom-file-label:active {
  opacity: 1;
}

/* Theme blending variables */
.dark-theme .custom-file-label {
  border-color: #4a5568;
}
.dark-theme .custom-file-label:active {
  background-color: rgba(255, 255, 255, 0.05);
}

.light-theme .custom-file-label {
  border-color: #cbd5e0;
}
.light-theme .custom-file-label:active {
  background-color: rgba(0, 0, 0, 0.03);
}


/* ==========================================================================
   🌊 WAVEFORM STYLE SCHEME
   ========================================================================== */
.track-waveform {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 32px;
  margin: 12px 0;
  width: 100%;
}

/* Individual Bar Layout settings */
.track-waveform .bar {
  display: inline-block;
  width: 3px;
  border-radius: 2px;
  background-color: currentColor;
  opacity: 0.4;
  transition: height 0.2s ease, opacity 0.2s ease;
}

/* Give the bars randomized default static heights like an audio signature */
.bar-1 { height: 8px; }
.bar-2 { height: 18px; }
.bar-3 { height: 28px; }
.bar-4 { height: 14px; }
.bar-5 { height: 24px; }
.bar-6 { height: 10px; }
.bar-7 { height: 16px; }

/* Dynamic Theme accent adjustment strings */
.dark-theme .track-waveform .bar {
  background-color: #63b3ed; /* Calm blue wave in dark mode */
}

.light-theme .track-waveform .bar {
  background-color: #3182ce; /* Deep ocean blue wave in light mode */
}

/* Optional Hover/Active Polish: Pulses the wave if you click it */
.track-row:active .bar {
  opacity: 0.8;
  animation: pulseWave 0.6s ease-in-out infinite alternate;
}

@keyframes pulseWave {
  0% { transform: scaleY(1); }
  100% { transform: scaleY(1.4); }
}
.track-select-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
}



/* ==========================================================================
   🌐 FIRST RUN INITIALIZATION UTILITY LAYOUT
   ========================================================================== */
.setup-screen-wrapper {
  min-height: 100vh;
  background-color: #121214; /* Default dark background on boot initialization */
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  box-sizing: border-box;
  color: #f7fafc;
}

.setup-card {
  background-color: #1a1a1e;
  border: 1px solid #2d3748;
  padding: 40px 30px;
  border-radius: 20px;
  max-width: 400px;
  width: 100%;
  text-align: center;
  box-shadow: 0 10px 25px rgba(0,0,0,0.3);
}

.setup-icon { font-size: 3.5rem; display: block; margin-bottom: 16px; }
.setup-card h2 { margin-bottom: 10px; font-size: 1.5rem; }
.setup-card p { opacity: 0.7; font-size: 0.95rem; margin-bottom: 24px; line-height: 1.5; }

.setup-input {
  width: 100%;
  padding: 14px;
  border-radius: 10px;
  border: 1px solid #4a5568;
  background-color: #2d3748;
  color: white;
  font-size: 1.1rem;
  text-align: center;
  margin-bottom: 16px;
  box-sizing: border-box;
}

.setup-submit-btn {
  width: 100%;
  padding: 14px;
  border-radius: 10px;
  border: none;
  background-color: #3182ce;
  color: white;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.setup-submit-btn:active { background-color: #2b6cb0; }
</style>