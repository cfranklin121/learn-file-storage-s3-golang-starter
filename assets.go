package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(mediaType string) string {
	base := make([]byte, 32)
	_, err := rand.Read(base)
	if err != nil {
		panic("failed to generate random bytes")
	}
	id := base64.RawURLEncoding.EncodeToString(base)

	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", id, ext)
}

func (cfg apiConfig) getObjectURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func (cfg apiConfig) getVideoAspectRatio(filePath string) (string, error) {
	var ptr bytes.Buffer

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &ptr

	err := cmd.Run()
	if err != nil {
		log.Printf("%s\n", err)
		return "", fmt.Errorf("Unable to run command: %s", err)
	}

	body := ptr.Bytes()

	type Aspect struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	var aspect Aspect
	err = json.Unmarshal(body, &aspect)
	if err != nil {
		return "", fmt.Errorf("Could not unmarshal: %s", err)
	}

	fmt.Println(aspect.Streams[0].Height)
	fmt.Println(aspect.Streams[0].Width)

	aspectRatio := float64(aspect.Streams[0].Width) / float64(aspect.Streams[0].Height)
	log.Println(aspectRatio)

	//16:9: 158.8 -> 159
	//9:16: 0.56  -> 1

	if aspectRatio > 1 {
		return "16:9", nil
	} else if aspectRatio < 1 {
		return "9:16", nil
	} else {
		return "other", nil
	}
}
