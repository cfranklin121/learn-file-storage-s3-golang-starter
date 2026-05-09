package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	//Chapter 3 Lesson 7 SDKs and S3
	//Step 3.1
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	//Step 3.2
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	//Step 3.3
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

	//Step 3.4
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Access denied", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

	//Step 3.5
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	//Step 3.6
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to get header", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusUnauthorized, "Invalid file type", err)
		return
	}

	//Step 3.7
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy file to temp file", err)
		return
	}

	//Step 3.8
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to reset temp file", err)
		return
	}

	//Step 3.9
	key := make([]byte, 32)
	rand.Read(key)
	str := base64.StdEncoding.EncodeToString(key)
	ext := strings.Split(mediaType, "/")
	keyString := str + "." + ext[1]
	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &keyString,
		Body:        tempFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading file to S3", err)
		return
	}

	//Step 3.10
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, keyString)
	video.VideoURL = &url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
