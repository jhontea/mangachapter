package checker

import (
	"project/mangachapter/internal/source"
)

// HasNewChapter menentukan apakah chapter yang diambil lebih baru dari yang tersimpan.
// Mengembalikan true jika ada chapter baru untuk dinotifikasi.
func HasNewChapter(storedNumValue float64, fetched *source.ChapterInfo) bool {
	if fetched == nil {
		return false
	}
	// Jika tidak ada chapter baseline yang tersimpan (0), anggap sebagai baru
	if storedNumValue == 0 {
		return true
	}
	return fetched.NumValue > storedNumValue
}