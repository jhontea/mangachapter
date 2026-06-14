package main

import (
	"fmt"
	"strings"

	"project/mangachapter/internal/source"
	"project/mangachapter/internal/storage"

	"github.com/spf13/cobra"
)

func newAddCmd(a *app) *cobra.Command {
	var urlFlag, idFlag string

	cmd := &cobra.Command{
		Use:   "add <source> <title>",
		Short: "Add a manga to the watchlist",
		Long:  `Add a manga to track. Fetches the latest chapter as baseline (no email sent).`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceName, title := args[0], args[1]

			src, ok := source.Get(sourceName)
			if !ok {
				return fmt.Errorf("unknown source %q; available: %s", sourceName, strings.Join(source.Available(), ", "))
			}

			// Determine the identifier for the manga
			mangaURL := urlFlag
			sourceID := idFlag

			if sourceName == "mangaplus" {
				// MangaPlus uses numeric ID
				if sourceID == "" {
					return fmt.Errorf("--id is required for mangaplus (numeric title ID)")
				}
				mangaURL = fmt.Sprintf("https://mangaplus.shueisha.co.jp/titles/%s", sourceID)
			} else {
				// Kiryuu and others use URL
				if mangaURL == "" {
					return fmt.Errorf("--url is required for %s", sourceName)
				}
				// Extract source ID from URL for kiryuu
				if sourceID == "" {
					sourceID = extractSourceID(sourceName, mangaURL)
				}
			}

			// Fetch baseline chapter
			fmt.Printf("Fetching latest chapter for %q...\n", title)
			ch, err := src.GetLatestChapter(a.context(), mangaURL)
			if err != nil {
				return fmt.Errorf("fetch latest chapter: %w", err)
			}

			// Save to database
			m := &storage.TrackedManga{
				Source:         sourceName,
				SourceID:       sourceID,
				Title:          title,
				URL:            mangaURL,
				LastChapter:    ch.Number,
				LastChapterNum: ch.NumValue,
			}

			if err := a.repo.AddManga(a.context(), m); err != nil {
				return fmt.Errorf("save manga: %w", err)
			}

			fmt.Printf("Added %q [%s] — latest: %s (ID: %d)\n", title, sourceName, ch.Number, m.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&urlFlag, "url", "", "manga page URL (required for kiryuu)")
	cmd.Flags().StringVar(&idFlag, "id", "", "manga title ID (required for mangaplus)")
	return cmd
}

// extractSourceID extracts the source-specific ID from a URL.
func extractSourceID(sourceName, mangaURL string) string {
	switch sourceName {
	case "kiryuu":
		// Extract slug from URL: https://v6.kiryuu.to/manga/slug/ → slug
		parts := strings.Split(strings.TrimRight(mangaURL, "/"), "/")
		for i, part := range parts {
			if part == "manga" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return mangaURL
}
