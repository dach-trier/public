package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dach-trier/env"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	var driveservice *drive.Service
	var err error

	if err = env.LoadFile(os.DirFS("."), ".env"); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			panic(err)
		}
	}

	ctx := context.Background()

	// ---
	// Drive
	// ---

	log.Printf("initializing Google Drive service ...")

	if driveservice, err = drive.NewService(ctx); err != nil {
		log.Printf("unable to initialize Google Drive service: %v\n", err)
		os.Exit(1)
	}

	if about, err := driveservice.About.Get().Fields("user(emailAddress)").Do(); err != nil {
		log.Printf("unable to fetch Google Drive account info: %v\n", err)
		os.Exit(1)
	} else if about == nil || about.User == nil || about.User.EmailAddress == "" {
		log.Printf("drive account is missing or incomplete\n")
		os.Exit(1)
	} else {
		log.Printf("successfully authenticated to Google Drive as %s\n", about.User.EmailAddress)
	}

	// ---
	// Router & Handlers
	// ---

	r := chi.NewRouter()
	r.Use(chi_middleware.Logger)

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if id == "" {
			http.Error(w, "missing file id", http.StatusBadRequest)
			return
		}

		res, err := driveservice.Files.Get(id).Download()

		if err != nil {
			var gerr *googleapi.Error

			if errors.As(err, &gerr) {
				switch gerr.Code {
				case 404:
					http.Error(w, "file not found", http.StatusNotFound)
					return
				case 403:
					http.Error(w, "access denied", http.StatusForbidden)
					return
				default:
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
		}

		defer res.Body.Close()

		if v := res.Header.Get("Content-Type"); v != "" {
			w.Header().Set("Content-Type", v)
		}

		if v := res.Header.Get("Content-Length"); v != "" {
			w.Header().Set("Content-Length", v)
		}

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		_, err = io.Copy(w, res.Body)

		if err != nil {
			log.Println("io.Copy: %v\n", err)
			return
		}
	})

	// ---
	// Server
	// ---

	port := env.GetIntInRange(os.Getenv, "PORT", 8080, 1, 65535)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: r,
	}

	log.Println("listening on :" + strconv.Itoa(port))
	log.Fatal(server.ListenAndServe())
}
