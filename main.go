package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/grafov/m3u8"
)

const testUrl string = "https://test-streams.mux.dev/x36xhzz/url_6/193039199_mp4_h264_aac_hq_7.m3u8"

const outpurDir string = "downloads"
const playlistName string = "playlist.m3u8"

func main() {
	// Set log level
	log.SetLevel(log.DebugLevel)

	// Create download directory if it doesn't exist
	if _, err := os.Stat(outpurDir); os.IsNotExist(err) {
		log.Debug("Creating download directory", "path", outpurDir)
		os.Mkdir(outpurDir, 0755)
	}

	// Download hls playlist
	plUrl, err := url.Parse(testUrl)
	if err != nil {
		log.Fatal("Failed to parse playlist url", "url", testUrl, "err", err)
	}

	resp, err := http.Get(plUrl.String())
	if err != nil {
		log.Fatal("Failed to download playlist", "url", testUrl, "err", err)
	}
	defer resp.Body.Close()

	// Write playlist to file
	out, err := os.Create(path.Join(outpurDir, playlistName))
	if err != nil {
		log.Fatal("Failed to create playlist file", "err", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal("Failed to write playlist file", "err", err)
	}

	// Open playlist file
	file, err := os.OpenFile(path.Join(outpurDir, playlistName), os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open playlist file", "err", err)
	}
	defer file.Close()

	// Parse playlist
	pl, listType, err := m3u8.DecodeFrom(file, true)
	if err != nil {
		log.Fatal("Failed to parse playlist", "err", err)
	}

	switch listType {
	case m3u8.MASTER:
		log.Fatal("Master playlists are not supported yet")
	case m3u8.MEDIA:
		masterpl := pl.(*m3u8.MediaPlaylist)

		for _, segment := range masterpl.Segments {
			// Skip segments with empty URI
			if segment == nil {
				continue
			}

			// Skip segments with absolute URI
			if strings.HasPrefix(segment.URI, "http") {
				log.Fatal("Absolute segment paths are not supported yet")
			}

			// Get absolute segment url from playlist url and segment path
			segUrl, err := getAbsoluteSegmentUrl(plUrl, segment.URI)
			if err != nil {
				log.Fatal("Failed to get absolute segment url", "err", err)
			}

			segDlPath := path.Join(outpurDir, segment.URI)

			// Download segment
			resp, err := http.Get(segUrl)
			if err != nil {
				log.Fatal("Failed to download segment", "url", segUrl, "err", err)
			}
			defer resp.Body.Close()

			// Write segment to file
			dir := filepath.Dir(segDlPath)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					log.Fatal("Failed to create segment directory", "path", dir, "err", err)
				}

			} else if err != nil {
				log.Fatal("Failed to stat segment directory", "path", dir, "err", err)
			}

			out, err := os.Create(segDlPath)
			if err != nil {
				log.Fatal("Failed to create segment file", "path", segDlPath, "err", err)
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Fatal("Failed to write segment file", "path", segDlPath, "err", err)
			}

			log.Info("Downloaded segment", "path", path.Join(outpurDir, segment.URI))
		}
	}

	log.Info("Downloaded playlist", "path", path.Join(outpurDir, playlistName))
}

// Returns absolute url of segment from playlist url and segment path
//
// Example: https://example.com/video/playlist.m3u8, /segment_0.ts -> https://example.com/video/segment_0.ts
func getAbsoluteSegmentUrl(plUrl *url.URL, segPath string) (string, error) {
	// Create url from segment path
	segUrl, err := url.Parse(segPath)
	if err != nil {
		return "", err
	}

	// Return absolute url
	return plUrl.ResolveReference(segUrl).String(), nil
}
