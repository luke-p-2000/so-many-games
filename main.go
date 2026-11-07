// So Many Games: a personal Windows app that tracks a Steam library and suggests what to
// play next. One window; closing it quits. See docs/spec for requirements and design.
package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"somanygames/internal/app"
	"somanygames/internal/steam"
)

//go:embed all:frontend/dist
var assets embed.FS

// version is set by production builds from build/windows/info.json (see build/windows/Taskfile.yml).
var version = "dev"

// lifecycle starts the launch sync once Wails is running.
type lifecycle struct{ core *app.Core }

func (l *lifecycle) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	l.core.Start(ctx)
	return nil
}

func (l *lifecycle) ServiceShutdown() error { return l.core.DB.Close() }

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	core, err := app.NewCore(filepath.Join(configDir, "SoManyGames"))
	if err != nil {
		log.Fatal(err)
	}
	core.Version = version
	steam.UserAgent = "SoManyGames/" + version

	wails := application.New(application.Options{
		Name:        "So Many Games",
		Description: "Track your Steam library and pick what to play next",
		Services: []application.Service{
			application.NewService(&lifecycle{core}),
			application.NewService(&app.SetupService{C: core}),
			application.NewService(&app.LibraryService{C: core}),
			application.NewService(&app.MetaService{C: core}),
			application.NewService(&app.RecService{C: core}),
			application.NewService(&app.WheelService{C: core}),
			application.NewService(&app.StatsService{C: core}),
			application.NewService(&app.SettingsService{C: core}),
			application.NewService(&app.DataService{C: core}),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: coversMiddleware(core.CoversDir()),
		},
	})
	core.Emit = func(name string, data any) { wails.Event.Emit(name, data) }

	window := wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "So Many Games",
		Width:            contentWidth,
		Height:           contentHeight,
		MinWidth:         1280,
		MinHeight:        720,
		Hidden:           true, // shown once sized to the screen
		BackgroundColour: application.NewRGB(17, 19, 24),
		URL:              "/",
	})
	wails.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if screen := wails.Screen.GetPrimary(); screen != nil {
			w, h := fitToScreen(screen.WorkArea.Width, screen.WorkArea.Height)
			window.SetSize(w, h)
		}
		window.Center()
		window.Show()
	})

	if err := wails.Run(); err != nil {
		log.Fatal(err)
	}
}

// The window opens at the width the layout is designed for: sidebar 240 + page 1400 +
// padding 88 px. It shrinks to fit smaller screens.
const (
	contentWidth  = 1728
	contentHeight = 1000
	screenMargin  = 40
)

func fitToScreen(workW, workH int) (int, int) {
	return min(contentWidth, workW-screenMargin), min(contentHeight, workH-screenMargin)
}

// coversMiddleware serves cached cover art from disk at /covers/<appid>.jpg.
func coversMiddleware(dir string) application.Middleware {
	files := http.StripPrefix("/covers/", http.FileServer(http.Dir(dir)))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/covers/") {
				files.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
