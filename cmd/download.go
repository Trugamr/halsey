package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/grafov/m3u8"
	"github.com/spf13/cobra"
)

func init() {
	downloadCmd.Flags().StringP("directory", "d", "playlist", "Output directory")

	rootCmd.AddCommand(downloadCmd)
}

var downloadCmd = &cobra.Command{
	Use:   "download [url]",
	Short: "Download a HLS stream",
	Long:  "Download a HLS stream from a given URL",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		// Check if input is a valid URL
		u, err := validateUrl(args[0])
		if err != nil {
			cobra.CheckErr("Invalid playlist URL")
		}

		log.Info("Downloading HLS stream", "url", u.String())

		// Collect flags
		directory := cmd.Flag("directory").Value.String()

		// Download playlist
		output := path.Join(directory, "index.m3u8")
		downloadPlaylist(u, output)

		log.Info("Download complete", "output", output)
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

// Download and save file to disk
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

// Download playlist and all referenced files
func downloadPlaylist(u *url.URL, output string) error {
	// Fetch playlist file
	resp, err := http.Get(u.String())
	if err != nil {
		log.Error("Failed to download playlist", "url", u.String(), "err", err)
		return err
	}

	// Parse playlist file
	pl, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		log.Error("Failed to parse playlist", "url", u.String(), "err", err)
		return err
	}

	switch listType {
	case m3u8.MASTER:
		pl := pl.(*m3u8.MasterPlaylist)

		// Download master playlist file
		log.Info("Downloading master playlist", "url", u.String())
		if err := downloadFile(u, output); err != nil {
			cobra.CheckErr("Failed to download master playlist")
		}

		for _, variant := range pl.Variants {
			if variant == nil {
				continue
			}

			vu, err := url.Parse(variant.URI)
			if err != nil {
				log.Error("Failed to parse variant url", "url", variant.URI, "err", err)
				cobra.CheckErr("Aborting download due to invalid variant url")
			}

			if vu.IsAbs() {
				log.Error("Absolute variant paths are not supported yet", "url", vu.String())
				cobra.CheckErr("Aborting download due to absolute variant url")
			} else {
				vu = u.ResolveReference(vu)
			}

			// Process available alternatives for variant
			for _, alt := range variant.Alternatives {
				if alt == nil {
					continue
				}

				au, err := url.Parse(alt.URI)
				if err != nil {
					log.Error("Failed to parse alternative url", "url", alt.URI, "err", err)
					cobra.CheckErr("Aborting download due to invalid alternative url")
				}

				if au.IsAbs() {
					log.Error("Absolute alternative paths are not supported yet", "url", au.String())
					cobra.CheckErr("Aborting download due to absolute alternative url")
				} else {
					au = u.ResolveReference(au)
				}

				downloadPlaylist(au, path.Join(path.Dir(output), alt.URI))
			}

			fmt.Println(output, variant.URI)
			downloadPlaylist(vu, path.Join(path.Dir(output), variant.URI))
		}

	case m3u8.MEDIA:
		pl := pl.(*m3u8.MediaPlaylist)

		// Download media playlist file
		log.Info("Downloading media playlist", "url", u.String())
		if err := downloadFile(u, output); err != nil {
			cobra.CheckErr("Failed to download media playlist")
		}

		for _, segment := range pl.Segments {
			if segment == nil {
				continue
			}

			su, err := url.Parse(segment.URI)
			if err != nil {
				log.Error("Failed to parse segment url", "url", segment.URI, "err", err)
				cobra.CheckErr("Aborting download due to invalid segment url")
			}

			if su.IsAbs() {
				fmt.Println(su)
				log.Error("Absolute segment paths are not supported yet", "url", su.String())
				cobra.CheckErr("Aborting download due to absolute segment url")
			} else {
				su = u.ResolveReference(su)

				// Download segment file
				log.Info("Downloading segment", "url", su.String())
				downloadFile(su, path.Join(path.Dir(output), segment.URI))
			}

		}
	default:
		cobra.CheckErr("Unknown playlist type")
	}

	return nil
}
