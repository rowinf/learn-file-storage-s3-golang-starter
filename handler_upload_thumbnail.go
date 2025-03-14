package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const MaxUploadSize = 10 << 20

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

	// TODO: implement the upload here

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing form", err)
		return
	}
	if file, fheader, err := r.FormFile("thumbnail"); err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing form", err)
		return
	} else {
		mediaType := fheader.Header["Content-Type"]
		if data, err := io.ReadAll(file); err != nil {
			respondWithError(w, http.StatusBadRequest, "Error parsing form", err)
			return
		} else {
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
					videoThumbnails[video.ID] = thumbnail{
						data:      data,
						mediaType: mediaType[0],
					}
					url := fmt.Sprintf("http://localhost:8091/api/thumbnails/%s", video.ID)
					video.ThumbnailURL = &url
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
