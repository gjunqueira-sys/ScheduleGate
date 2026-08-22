package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

type FileService struct{}

func (f *FileService) getWindow() application.Window {
	app := application.Get()
	windows := app.Window.GetAll()
	if len(windows) > 0 {
		return windows[0]
	}
	return nil
}

func (f *FileService) OpenScheduleFile() (string, error) {
	app := application.Get()
	dialog := app.Dialog.OpenFile().
		SetTitle("Select Schedule File").
		AddFilter("Schedule Files (*.xlsx, *.csv)", "*.xlsx;*.csv").
		AddFilter("All Files (*.*)", "*.*")
	if w := f.getWindow(); w != nil {
		dialog.AttachToWindow(w)
	}
	return dialog.PromptForSingleSelection()
}

func (f *FileService) OpenRulesFile() (string, error) {
	app := application.Get()
	dialog := app.Dialog.OpenFile().
		SetTitle("Select Rules File").
		AddFilter("YAML Files (*.yaml, *.yml)", "*.yaml;*.yml").
		AddFilter("All Files (*.*)", "*.*")
	if w := f.getWindow(); w != nil {
		dialog.AttachToWindow(w)
	}
	return dialog.PromptForSingleSelection()
}

func (f *FileService) SaveFileDialog(title, defaultName string) (string, error) {
	app := application.Get()
	dialog := app.Dialog.SaveFile().
		SetMessage(title).
		SetFilename(defaultName)
	if w := f.getWindow(); w != nil {
		dialog.AttachToWindow(w)
	}
	return dialog.PromptForSingleSelection()
}
