package app

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"

	"somanygames/internal/transfer"
)

// DataService handles backup, export and import (B1, X1) through native file dialogs.
type DataService struct {
	C *Core

	pending *transfer.File // parsed import awaiting confirmation
}

// ErrCancelled is returned when the owner closes a file dialog.
var ErrCancelled = errors.New("cancelled")

// Backup asks where to save and writes a copy of the database.
func (s *DataService) Backup() (string, error) {
	path, err := s.saveDialog("somanygames-backup.db", "SQLite database", "*.db")
	if err != nil {
		return "", err
	}
	return path, transfer.Backup(s.C.DB, path)
}

// Export asks where to save and writes the JSON export.
func (s *DataService) Export() (string, error) {
	path, err := s.saveDialog("somanygames-export.json", "JSON", "*.json")
	if err != nil {
		return "", err
	}
	f, err := transfer.Export(s.C.DB, s.C.now())
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, b, 0o644)
}

// PreviewImport asks for a file and returns what importing it would change.
func (s *DataService) PreviewImport() (transfer.Summary, error) {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Import So Many Games data").
		AddFilter("JSON", "*.json").
		PromptForSingleSelection()
	if err != nil {
		return transfer.Summary{}, err
	}
	if path == "" {
		return transfer.Summary{}, ErrCancelled
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return transfer.Summary{}, err
	}
	f, err := transfer.Parse(data)
	if err != nil {
		return transfer.Summary{}, err
	}
	sum, err := transfer.Summarise(s.C.DB, f)
	if err != nil {
		return sum, err
	}
	s.pending = &f
	return sum, nil
}

// ConfirmImport replaces the current data with the previewed file.
func (s *DataService) ConfirmImport() error {
	if s.pending == nil {
		return errors.New("choose a file to import first")
	}
	f := *s.pending
	s.pending = nil
	if err := transfer.Apply(s.C.DB, f, s.C.now()); err != nil {
		return err
	}
	s.C.Emit(EventLibraryChanged, nil)
	return nil
}

// OpenStorePage opens a game's Steam store page in the default browser.
func (s *DataService) OpenStorePage(appid int) error {
	return application.Get().Browser.OpenURL("https://store.steampowered.com/app/" + strconv.Itoa(appid))
}

func (s *DataService) saveDialog(name, label, pattern string) (string, error) {
	path, err := application.Get().Dialog.SaveFile().
		SetFilename(name).
		AddFilter(label, pattern).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", ErrCancelled
	}
	return path, nil
}
