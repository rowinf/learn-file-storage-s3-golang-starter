package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	MaxVideoUploadSize = 1 << 30
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading video for video", videoID, "by user", userID)
	// implement the upload here
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
			file, header, err := r.FormFile("video")
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "error getting file", err)
				return
			}
			defer file.Close()
			mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
			if mediaType != "video/mp4" || err != nil {
				respondWithError(w, http.StatusBadRequest, "Error parsing Content-Type", err)
				return
			}
			f, err := os.CreateTemp("", "tubely-upload.mp4")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(f.Name())
			reader := io.Reader(file)
			if _, err := io.Copy(f, reader); err != nil {
				log.Fatal(err)
			}
			f.Seek(0, io.SeekStart)
			bucket := os.Getenv("S3_BUCKET")
			key := make([]byte, 32)
			rand.Read(key)
			dst := make([]byte, base64.RawURLEncoding.EncodedLen(len(key)))
			base64.RawURLEncoding.Encode(dst, key)
			fileName := string(dst) + ".mp4"
			region := os.Getenv("S3_REGION")
			input := s3.PutObjectInput{Bucket: &bucket, Key: &fileName, ContentType: &mediaType, Body: f}
			if _, err := cfg.s3Client.PutObject(context.Background(), &input); err != nil {
				log.Fatal(err)
			}

			url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, fileName)
			video.VideoURL = &url
			if err := cfg.db.UpdateVideo(video); err != nil {
				respondWithError(w, http.StatusBadRequest, "Error saving video", err)
				return
			}
			if vid, err := cfg.db.GetVideo(video.ID); err != nil {
				respondWithError(w, http.StatusBadRequest, "Error saving video", err)
			} else {
				respondWithJSON(w, http.StatusOK, vid)
			}
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}
}
