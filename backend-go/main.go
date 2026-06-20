package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"path/filepath"
"embed"
"io/fs"
	_ "github.com/mattn/go-sqlite3"
)

// 🌟 1. Tell Go to bake the "dist" folder directly into the binary
//go:embed dist/*
var staticFS embed.FS


var db *sql.DB
var audioDir string // Global variable to hold our dynamic audio path

// Track represents our database structure
type Track struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`    // Added 'string' here
	Duration string `json:"duration"` // Added 'string' here
	URL      string `json:"url"`
}


func main() {
	// 1. Define command-line options with smart defaults
	port := flag.String("port", "8080", "Port to run the server on")
	flag.StringVar(&audioDir, "audio", "./audio", "Directory path where audio files are stored")
	dbPath := flag.String("db", "./meditations.db", "Path to the SQLite database file")
	flag.Parse()

	var err error
	// 2. Open SQLite using the dynamic path cleanly (Only once!)
	db, err = sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize connection string: %v", err)
	}
	defer db.Close()

	// Use db.Exec directly for simple table creations and catch errors!
	createTableQuery := `CREATE TABLE IF NOT EXISTS tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		duration TEXT,
		filename TEXT
		);`

		_, err = db.Exec(createTableQuery)
		if err != nil {
			log.Fatalf("❌ Termux couldn't build database tables: %v", err)
		}
		log.Println("✅ SQLite database and tables verified successfully!")

		// Ensure the designated audio directory exists
		os.MkdirAll(audioDir, os.ModePerm)

		// 3. Setup Routes on the Default Router
		http.HandleFunc("/api/tracks/", handleTracks) // 🌟 KEPT: This handles GET, PUT, and DELETE cleanly!
		http.HandleFunc("/api/upload", handleUpload)

		// 🌟 Explicitly register the verification route handler block
		http.HandleFunc("/api/verify-admin", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var requestData struct {
					Password string `json:"password"`
				}
				if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error": "Invalid payload"}`))
					return
				}

				actualAdminPass := os.Getenv("MEDITATION_ADMIN_PASS")
				if actualAdminPass == "" {
					actualAdminPass = "jo11"
				}

				w.Header().Set("Content-Type", "application/json")
				if requestData.Password == actualAdminPass {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"success": true}`))
				} else {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error": "Incorrect password"}`))
				}
				return
			}
		})

		// Serve files with streaming headers out of audio directory
		fileServer := http.FileServer(http.Dir(audioDir))
		http.HandleFunc("/audio/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Range")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.StripPrefix("/audio/", fileServer).ServeHTTP(w, r)
		})

		// Get the sub-filesystem for the embedded Vue "dist" directory
		distFiles, err := fs.Sub(staticFS, "dist")
		if err != nil {
			log.Fatalf("❌ Failed to read embedded static files: %v", err)
		}

		// Serve the static Vue files on the root "/" route
		http.Handle("/", http.FileServer(http.FS(distFiles)))

		// 4. WRAP THE DEFAULT ROUTER INSIDE A GLOBAL CORS MIDDLEWARE
		globalRouterWithCORS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Range")

			// Intercept browser preflight OPTIONS request instantly
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Pass normal traffic down to your registered routes
			http.DefaultServeMux.ServeHTTP(w, r)
		})

		// 5. Start listening using our corsWrapped middleware layout wrapper
		log.Printf("Meditation backend listening on all interfaces (0.0.0.0) at port %s", *port)
		log.Printf("Serving audio assets from: %s", audioDir)
		log.Fatal(http.ListenAndServe("0.0.0.0:"+*port, globalRouterWithCORS))
}


// Handles fetching tracks from SQLite, and deleting them safely
func handleTracks(w http.ResponseWriter, r *http.Request) {
    // 1. Setup fundamental global CORS Headers
    w.Header().Set("Access-Control-Allow-Origin", "*")

    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")

    // 2. Handle browser Preflight OPTIONS requests cleanly
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }



	if strings.HasPrefix(r.URL.Path, "/api/verify-admin") {
    // 1. Set global CORS headers for this route immediately
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    // 2. Handle the Preflight (OPTIONS) request instantly with a 200 OK status
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }

    // 3. Process the actual POST request
    if r.Method == http.MethodPost {
        var requestData struct {
            Password string `json:"password"`
        }
        if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusBadRequest)
            w.Write([]byte(`{"error": "Invalid payload"}`))
            return
        }

        // Pull the password safely from the system environment keys
        actualAdminPass := os.Getenv("MEDITATION_ADMIN_PASS")
        if actualAdminPass == "" {
            actualAdminPass = "jo11" // Safe fallback default string
        }

        w.Header().Set("Content-Type", "application/json")
        if requestData.Password == actualAdminPass {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{"success": true}`))
        } else {
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"error": "Incorrect password"}`))
        }
        return
    }
}




// 📝 HANDLE THE PUT (UPDATE) METHOD
    if r.Method == http.MethodPut {
        // Parse the Track ID from the end of the URL path layout safely
        pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
        trackID := pathParts[len(pathParts)-1]

        if trackID == "" {
            http.Error(w, `{"error": "Missing track ID parameter"}`, http.StatusBadRequest)
            return
        }

        // Decode incoming JSON fields (Title and Duration updates)
        var updatedData struct {
            Title    string `json:"title"`
            Duration string `json:"duration"`
        }
        if err := json.NewDecoder(r.Body).Decode(&updatedData); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        // Update the metadata directly inside SQLite database storage
        _, err := db.Exec("UPDATE tracks SET title = ?, duration = ? WHERE id = ?", 
            updatedData.Title, updatedData.Duration, trackID)
        if err != nil {
            http.Error(w, `{"error": "Database update failed"}`, http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"message": "Track metadata updated successfully"}`))
        return
    }


	

    // 3. 🗑️ HANDLE THE DELETE REQ METHOD
    if r.Method == http.MethodDelete {
        // Strip the "/api/tracks/" route prefix to extract the raw ID string
        trackID := r.URL.Path[len("/api/tracks/"):]
        if trackID == "" {
            http.Error(w, `{"error": "Missing track ID parameter"}`, http.StatusBadRequest)
            return
        }

        // Fetch the file path directory out of the database to remove it from disk storage
        var filename string
        err := db.QueryRow("SELECT filename FROM tracks WHERE id = ?", trackID).Scan(&filename)
        if err == nil && filename != "" {
            // Delete file physically from your audioDir path
            os.Remove(filepath.Join(audioDir, filename))
        }

        // Delete row entry out of SQLite 
        _, err = db.Exec("DELETE FROM tracks WHERE id = ?", trackID)
        if err != nil {
            http.Error(w, `{"error": "Database deletion failed"}`, http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"message": "Track deleted successfully"}`))
        return
    }

    // 4. 🎵 HANDLE THE GET REQ METHOD
    if r.Method == http.MethodGet {
        w.Header().Set("Content-Type", "application/json")

        rows, err := db.Query("SELECT id, title, duration, filename FROM tracks")
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var tracks []Track
        for rows.Next() {
            var t Track
            var filename string
            if err := rows.Scan(&t.ID, &t.Title, &t.Duration, &filename); err != nil {
                continue
            }
            
            // Dynamic URL constructor layout using your r.Host approach
            scheme := "http://"
            if r.TLS != nil { 
                scheme = "https://" 
            }
            t.URL = scheme + r.Host + "/audio/" + filename
            
            tracks = append(tracks, t)
        }
        json.NewEncoder(w).Encode(tracks)
        return
    }

    // Reject unsupported methods gracefully
    http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
}

// Handles saving uploaded file and writing to database
func handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(50 << 20)
	title := r.FormValue("title")
	duration := r.FormValue("duration")

	file, handler, err := r.FormFile("audioFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Use our globally assigned configuration path
	localFilePath := filepath.Join(audioDir, handler.Filename)
	targetFile, err := os.OpenFile(localFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, "Unable to save file on server", http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()
	io.Copy(targetFile, file)

	stmt, _ := db.Prepare("INSERT INTO tracks (title, duration, filename) VALUES (?, ?, ?)")
	_, err = stmt.Exec(title, duration, handler.Filename)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Upload successful!"}`))
}

func handleStreamAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	filePath := filepath.Join(".", r.URL.Path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Audio file not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filePath)
}