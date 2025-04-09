package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"

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
			tempfile, err := os.CreateTemp("", "tubely-upload.mp4")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(tempfile.Name())
			reader := io.Reader(file)
			if _, err := io.Copy(tempfile, reader); err != nil {
				log.Fatal(err)
			}
			tempfile.Seek(0, io.SeekStart)

			processedFileName, err := processVideoForFastStart(tempfile.Name())
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "couldnt process for fast start", err)
				return
			}
			processedFile, err := os.Open(processedFileName)
			defer os.Remove(processedFileName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "couldnt open processed file", err)
				return
			}
			bucket := os.Getenv("S3_BUCKET")
			key := make([]byte, 32)
			rand.Read(key)
			dst := make([]byte, base64.RawURLEncoding.EncodedLen(len(key)))
			base64.RawURLEncoding.Encode(dst, key)
			aspectRatio, err := GetVideoAspectRatio(tempfile.Name())
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "couldnt get aspect ratio", err)
				return
			}
			prefix := "other"
			if aspectRatio == "16:9" {
				prefix = "landscape"
			} else if aspectRatio == "9:16" {
				prefix = "portrait"
			}
			fileName := bucket + "," + prefix + "/" + string(dst) + ".mp4"
			region := os.Getenv("S3_REGION")
			input := s3.PutObjectInput{Bucket: &bucket, Key: &fileName, ContentType: &mediaType, Body: processedFile}
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
				presignedVideo, err := cfg.dbVideoToSignedVideo(vid)
				if err != nil {
					respondWithError(w, http.StatusBadRequest, "couldnt sign video url", err)
					return
				}
				respondWithJSON(w, http.StatusOK, presignedVideo)
			}
			if err := tempfile.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}
}

type Aspect struct {
	DisplayAspectRatio string `json:"display_aspect_ratio"`
}

type Results struct {
	Streams []Aspect
}

func GetVideoAspectRatio(filepath string) (string, error) {
	ratio := ""
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return ratio, err
	}

	var results Results
	json.Unmarshal(buf.Bytes(), &results)
	ratio = results.Streams[0].DisplayAspectRatio
	return ratio, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFileName := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFileName, "-y", "-v", "quiet")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return outputFileName, err
	}
	return outputFileName, nil
}
