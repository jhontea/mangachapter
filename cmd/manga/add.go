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
		Short: "Tambahkan manga ke daftar pantau",
		Long:  `Tambahkan manga untuk dilacak. Mengambil chapter terbaru sebagai baseline (tanpa notifikasi).`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceName, title := args[0], args[1]

			src, ok := source.Get(sourceName)
			if !ok {
				return fmt.Errorf("sumber tidak dikenal %q; tersedia: %s", sourceName, strings.Join(source.Available(), ", "))
			}

			// Tentukan identifier untuk manga
			mangaURL := urlFlag
			sourceID := idFlag

			if sourceName == "mangaplus" {
				// MangaPlus menggunakan ID numerik
				if sourceID == "" {
					return fmt.Errorf("--id wajib diisi untuk mangaplus (ID judul numerik)")
				}
				mangaURL = fmt.Sprintf("https://mangaplus.shueisha.co.jp/titles/%s", sourceID)
			} else {
				// Kiryuu dan lainnya menggunakan URL
				if mangaURL == "" {
					return fmt.Errorf("--url wajib diisi untuk %s", sourceName)
				}
				// Ekstrak source ID dari URL untuk kiryuu
				if sourceID == "" {
					sourceID = extractSourceID(sourceName, mangaURL)
				}
			}

			// Ambil chapter baseline
			fmt.Printf("Mengambil chapter terbaru untuk %q...\n", title)
			ch, err := src.GetLatestChapter(a.context(), mangaURL)
			if err != nil {
				return fmt.Errorf("ambil chapter terbaru: %w", err)
			}

			// Simpan ke database
			m := &storage.TrackedManga{
				Source:         sourceName,
				SourceID:       sourceID,
				Title:          title,
				URL:            mangaURL,
				LastChapter:    ch.Number,
				LastChapterNum: ch.NumValue,
			}

			if err := a.repo.AddManga(a.context(), m); err != nil {
				return fmt.Errorf("simpan manga: %w", err)
			}

			fmt.Printf("Menambahkan %q [%s] — terbaru: %s (ID: %d)\n", title, sourceName, ch.Number, m.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&urlFlag, "url", "", "URL halaman manga (wajib untuk kiryuu)")
	cmd.Flags().StringVar(&idFlag, "id", "", "ID judul manga (wajib untuk mangaplus)")
	return cmd
}

// extractSourceID mengekstrak ID spesifik sumber dari URL.
func extractSourceID(sourceName, mangaURL string) string {
	switch sourceName {
	case "kiryuu":
		// Ekstrak slug dari URL: https://v6.kiryuu.to/manga/slug/ → slug
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