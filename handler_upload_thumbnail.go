package main

import (
	"fmt"
	"net/http"
	"io"
	"os"
	"mime"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "the given id is not valid", err)
		return
	}

	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "video not found", err)
		return
	}
	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "this is not your video", err)
		return
	}
	mediatype, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "the header is incorrect", err)
		return
	}
	splitedContentType := strings.Split(mediatype, "/")
	fileExtension := splitedContentType[1]
	if fileExtension != "jpeg" && fileExtension != "png" {
		respondWithError(w, http.StatusBadRequest, "not allowed to upload this file type", err)
		return
	}
	thumbnailFile := fmt.Sprintf("%s.%s", videoIDString, fileExtension)
	thumbnailFilePath := filepath.Join(cfg.assetsRoot, thumbnailFile)
	newFile, err := os.Create(thumbnailFilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while creating", err)
		return
	}

	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while copying", err)
		return
	}

	url := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbnailFile)
	video.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "something went wrong while creating", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
