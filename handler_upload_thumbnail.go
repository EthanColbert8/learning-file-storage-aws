package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory int64 = 10 << 20

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

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Improper form body", err)
		return
	}

	newFile, newFileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read file data", err)
		return
	}
	defer newFile.Close()

	fileType := newFileHeader.Header.Get("Content-Type")
	if fileType == "" {
		respondWithError(w, http.StatusBadRequest, "Improper file metadata", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video data", err)
		return
	}

	// validate that authenticated user owns the video
	if videoMetadata.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not owner of video", nil)
		return
	}

	fileExtension, err := readContentType(fileType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid file metadata", err)
		return
	}

	newFilePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoID, fileExtension))

	newFileCopy, err := os.Create(newFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create local thumbnail file", err)
		return
	}
	defer newFileCopy.Close()

	_, err = io.Copy(newFileCopy, newFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save thumbnail data", err)
		return
	}

	newURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoID, fileExtension)

	videoMetadata.ThumbnailURL = &newURL
	cfg.db.UpdateVideo(videoMetadata)

	respondWithJSON(w, http.StatusOK, videoMetadata)
}

/*
 * A helper function to read the "Content-Type" header on form data
 * into a usable file extension.
 */
func readContentType(ts string) (string, error) {
	parts := strings.Split(ts, "/")

	if len(parts) != 2 {
		return "", fmt.Errorf("Not a valid image content type: %s", ts)
	}

	if parts[0] != "image" {
		return "", fmt.Errorf("Not a valid image content type: %s", ts)
	}

	return parts[1], nil
}
