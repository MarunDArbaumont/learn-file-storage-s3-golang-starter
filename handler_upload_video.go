package main

import (
	"net/http"
	"crypto/rand"
	"mime"
	"strings"
	"os"
	"io"
	"encoding/hex"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"fmt"

	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
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

	fmt.Println("uploading video", videoID, "by user", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "video not found", err)
		return
	}
	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "this is not your video", err)
		return
	}

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "max memory", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	
	mediatype, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "the header is incorrect", err)
		return
	}
	splitedContentType := strings.Split(mediatype, "/")
	fileExtension := splitedContentType[1]
	if fileExtension != "mp4" {
		respondWithError(w, http.StatusBadRequest, "not allowed to upload this file type", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "not allowed to upload this file type", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while copying", err)
		return
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while reseting the temp file", err)
		return
	}

	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	key := hex.EncodeToString(randomBytes) + "." + fileExtension

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(key),
		Body: tempFile,
		ContentType: aws.String(mediatype),
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while reseting the temp file", err)
		return
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	video.VideoURL = &url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while creating", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
