const DB_NAME = 'MeditationOfflineDB';
const STORE_NAME = 'audio_tracks';

// Open (or create) the IndexedDB database
const openDB = () => {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    
    request.onupgradeneeded = (e) => {
      const db = e.target.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    };
    
    request.onsuccess = (e) => resolve(e.target.result);
    request.onerror = (e) => reject(e.target.error);
  });
};

export const offlineStorage = {
  // Save an audio blob locally indexed by its track ID
  async saveTrack(trackId, blob) {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, 'readwrite');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.put(blob, String(trackId));
      
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  },

  // Retrieve an audio blob by track ID
  async getTrack(trackId) {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, 'readonly');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.get(String(trackId));
      
      request.onsuccess = () => resolve(request.result); // Returns Blob or undefined
      request.onerror = () => reject(request.error);
    });
  }
};