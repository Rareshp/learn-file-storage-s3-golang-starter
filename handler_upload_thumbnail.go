package main

import (
	"fmt"
	"io"
	"net/http"
  "os"
  "path/filepath"
  "mime"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	_ "github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

// 10 MB
const maxMemory int = 10 << 20;

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

  err = r.ParseMultipartForm(int64(maxMemory))
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Something went wrong parsing form", err)
    return
  }
  // the above is needed first
  imageData, header, err := r.FormFile("thumbnail");
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Something went wrong parsing form", err)
    return
  }
  defer imageData.Close()

  // example "image/png"
  mediaType := header.Header.Get("Content-Type");
  if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
    return
  }
  fileExtensions, err := mime.ExtensionsByType(mediaType)
  if err != nil || len(fileExtensions) == 0 {
      respondWithError(w, http.StatusInternalServerError, "Could not compute file extensions", err)
      return
  }
  // this has a period already
  fileExtension := fileExtensions[0]
  if fileExtension != ".jpeg" && fileExtension != ".png" {
    respondWithError(w, http.StatusUnsupportedMediaType, "File type not supported. Use jpeg or png.", err)
    return
  }

  videoMetadata, err := cfg.db.GetVideo(videoID);
  if err != nil {
    respondWithError(w, http.StatusUnauthorized, "Couldn't find video", err);
    return
  }
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

  // the write file needs the bytes form
  // /assetsRoot/<videoID>.<file_extension>
  fullPath2File := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s%s", videoIDString, fileExtension) );

  // do not use os.WriteFile because the file is multipart; might be in process
  thumbFile, err := os.Create(fullPath2File);
  if err != nil {
    txt := fmt.Sprintf("Could not create thumbnail file: %s", fullPath2File)
    respondWithError(w, http.StatusInternalServerError, txt, err)
    return
  }
  defer thumbFile.Close();

  _, err = io.Copy(thumbFile, imageData)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Could not save thumbnail file", err)
    return
  }

  thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s%s", cfg.port, videoIDString, fileExtension);

  videoMetadata.ThumbnailURL = &thumbnailURL;

  err = cfg.db.UpdateVideo(videoMetadata);
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
    return
  }

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
