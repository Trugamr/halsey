package cmd

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/grafov/m3u8"
	"github.com/spf13/cobra"
)

const defaultPlaylistFilename string = "index.m3u8"

func init() {
	downloadCmd.Flags().StringP("output", "o", "./playlist/"+defaultPlaylistFilename, "Output path for downloaded files")

	rootCmd.AddCommand(downloadCmd)
}

var downloadCmd = &cobra.Command{
	Use:   "download [url]",
	Short: "Download a HLS stream",
	Long:  "Download a HLS stream from a given URL",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		// Check if input is a valid URL
		plUrl, err := validateUrl(args[0])
		if err != nil {
			cobra.CheckErr("Invalid playlist URL")
		}

		log.Info("Downloading HLS stream", "url", plUrl.String())

		// Collect flags
		output := cmd.Flag("output").Value.String()

		// Download and save playlist
		plPath := getPlaylistFilePath(output)
		err = downloadFile(plUrl, plPath)
		if err != nil {
			cobra.CheckErr("Failed to download playlist")
		}

		// Read and parse playlist
		f, err := os.Open(plPath)
		if err != nil {
			log.Error("Failed to open playlist file", "path", plPath, "err", err)
			cobra.CheckErr("Could not open playlist file")
		}
		defer f.Close()

		pl, listType, err := m3u8.DecodeFrom(f, true)
		if err != nil {
			log.Error("Failed to parse playlist", "path", plPath, "err", err)
			cobra.CheckErr("Invalid playlist file")
		}

		switch listType {
		case m3u8.MASTER:
			cobra.CheckErr("Master playlist files are not supported yet")
		case m3u8.MEDIA:
			mediapl := pl.(*m3u8.MediaPlaylist)

			for _, segment := range mediapl.Segments {
				// Ignore segments with no uri
				if segment == nil {
					continue
				}

				// Download and save segment
				segUrl, err := url.Parse(segment.URI)
				if err != nil {
					log.Error("Failed to parse segment url", "url", segment.URI, "err", err)
					cobra.CheckErr("Aborting download due to invalid segment url")
				}

				// Check if segment url is absolute
				// TODO: Support absolute segment urls
				if segUrl.IsAbs() {
					log.Error("Absolute segment paths are not supported yet", "url", segUrl.String())
					cobra.CheckErr("Aborting download due to absolute segment url")
				}

				// Get absolute segment url from playlist url and segment path
				segResolvedUrl := plUrl.ResolveReference(segUrl)
				log.Info("Downloading segment", "url", segResolvedUrl)

				segPath := getDownloadPath(getPlaylistDirectoryPath(plPath), segment.URI)

				err = downloadFile(segResolvedUrl, segPath)
				if err != nil {
					log.Error("Failed to download segment", "url", segResolvedUrl, "err", err)
					cobra.CheckErr("Aborting download due to failed segment download")
				}

				log.Info("Downloaded segment", "path", segPath)
			}
		default:
			cobra.CheckErr("Unknown playlist type")
		}

		log.Info("Download complete", "path", plPath)
	},
}

// Check if input is a valid URL
//
// Returns parsed url
func validateUrl(input string) (u *url.URL, err error) {
	u, err = url.ParseRequestURI(input)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Download and save file
//
// Returns path to downloaded file
func downloadFile(u *url.URL, path string) error {
	resp, err := http.Get(u.String())
	if err != nil {
		log.Error("Failed to download file", "url", u.String(), "err", err)
		return err
	}
	defer resp.Body.Close()

	// Create output directory if it doesn't exist
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Error("Failed to create output directory", "path", dir, "err", err)
			return err
		}
	} else if err != nil {
		log.Error("Failed to stat output directory", "path", dir, "err", err)
		return err
	}

	// Create output file
	output, err := os.Create(path)
	if err != nil {
		log.Error("Failed to create output file", "path", path, "err", err)
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		log.Error("Failed to write output file", "path", path, "err", err)
		return err
	}

	return nil
}

func getPlaylistFilePath(output string) string {
	// If output is a directory, append default filename
	if filepath.Ext(output) == "" || filepath.Ext(output) == "." {
		return getDownloadPath(output, defaultPlaylistFilename)
	}
	return output
}

// Takes playlist output path and returns directory path
func getPlaylistDirectoryPath(path string) string {
	return filepath.Dir(path)
}

func getDownloadPath(dir, path string) string {
	return filepath.Join(dir, path)
}
