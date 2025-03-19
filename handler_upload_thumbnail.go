package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	MaxUploadSize = 10 << 20
	urlRoot       = "http://localhost:8091/"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// implement the upload here

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing form", err)
		return
	}
	if file, fheader, err := r.FormFile("thumbnail"); err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing form", err)
		return
	} else {
		mediaType, _, mediaError := mime.ParseMediaType(fheader.Header.Get("Content-Type"))
		if mediaType != "image/png" && mediaType != "image/jpeg" || mediaError != nil {
			respondWithError(w, http.StatusBadRequest, "Error parsing Content-Type", err)
			return
		}
		reader := io.Reader(file)
		//
		if video, err := cfg.db.GetVideo(videoID); err != nil {
			respondWithError(w, http.StatusBadRequest, "Video Not Found", err)
			return
		} else {
			if user, err := cfg.db.GetUser(userID); err != nil {
				http.Error(w, "User Not Found", http.StatusBadRequest)
				return
			} else {
				if user.ID != video.UserID {
					respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
					return
				}
				// authorized
				if extensions, err := mime.ExtensionsByType(mediaType); err != nil {
					respondWithError(w, http.StatusBadRequest, "Error parsing mediaType", err)
					return
				} else {
					key := make([]byte, 32)
					rand.Read(key)
					dst := make([]byte, base64.RawURLEncoding.EncodedLen(len(key)))
					base64.RawURLEncoding.Encode(dst, key)
					fileName := string(dst) + extensions[0]
					// fileName := videoIDString + extensions[0]
					filePath := urlRoot + filepath.Join(cfg.assetsRoot, fileName)
					systemPath := filepath.Join(cfg.assetsRoot, fileName)
					if localFile, err := os.Create(systemPath); err != nil {
						respondWithError(w, http.StatusBadRequest, "error creating file", err)
						return
					} else {
						if _, err := io.Copy(localFile, reader); err != nil {
							respondWithError(w, http.StatusBadRequest, "error copying file content", err)
							return
						} else {
							video.ThumbnailURL = &filePath

							if err := cfg.db.UpdateVideo(video); err != nil {
								respondWithError(w, http.StatusBadRequest, "Error saving video", err)
								return
							}
							if vid, err := cfg.db.GetVideo(video.ID); err != nil {
								respondWithError(w, http.StatusBadRequest, "Error saving video", err)
							} else {
								respondWithJSON(w, http.StatusOK, vid)
							}
						}
					}
				}
			}
		}
	}
}
