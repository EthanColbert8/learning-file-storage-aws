package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxUploadSize int64 = 1 << 30 // 1 GB

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	_ = http.MaxBytesReader(w, r.Body, maxUploadSize) // not sure if this is correct

	videoID, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unable to find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to validate JWT", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Unable to find video data", err)
		return
	}

	// validate that authenticated user owns the video object
	if videoMetadata.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not owner of video", nil)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Improper form body", err)
		return
	}

	newFile, newFileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read file data", err)
		return
	}
	defer newFile.Close()

	fileType := newFileHeader.Header.Get("Content-Type")
	if fileType == "" {
		respondWithError(w, http.StatusBadRequest, "Improper file metadata", nil)
		return
	}

	err = readVideoContentType(fileType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid file metadata", err)
		return
	}
	var fileExtension string = "mp4"

	tempFile, err := os.CreateTemp("", fmt.Sprintf("tubely_upload_*.%s", fileExtension))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create server-side temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, newFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload file", err)
		return
	}
	tempFile.Seek(0, io.SeekStart) // resets file pointer to beginning for reading

	// read a random name for the new file
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	newFileName := base64.RawURLEncoding.EncodeToString(randBytes)

	newFileKey := fmt.Sprintf("%s.%s", newFileName, fileExtension)
	contentMimeType := fmt.Sprintf("video/%s", fileExtension)

	putObjectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &newFileKey,
		Body:        tempFile,
		ContentType: &contentMimeType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &putObjectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to store video file", err)
		return
	}

	newURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, newFileKey)
	videoMetadata.VideoURL = &newURL
	cfg.db.UpdateVideo(videoMetadata)

	respondWithJSON(w, http.StatusOK, videoMetadata)
}

/*
 * A helper function to validate the content type of the video form file.
 * Currently, we only accept MP4 video files.
 *
 * To allow for others, modify this function to also return a string that is
 * the file extension, and have it function like the image version
 * in handler_upload_thumbnail.go. Then modify the call to it to store the
 * string output in the fileExtension variable. Everything will just work.
 */
func readVideoContentType(ts string) error {
	mediaType, _, err := mime.ParseMediaType(ts)
	if err != nil {
		return fmt.Errorf("found invalid media type: %w", err)
	}

	if mediaType != "video/mp4" {
		return fmt.Errorf("only MP4 video files are allowed, found: %s", mediaType)
	}

	return nil
}
