package main

import (
	"embed"
	"os"

	"github.com/gjunqueira-sys/ScheduleGate/desktop/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	icon, _ := os.ReadFile("build/appicon.png")

	app := application.New(application.Options{
		Name:        "ScheduleGate",
		Description: "DCMA 14-Point Schedule Assessment Tool",
		Icon:        icon,

		Services: []application.Service{
			application.NewService(&services.FileService{}),
			application.NewService(&services.AssessService{}),
			application.NewService(&services.CompareService{}),
			application.NewService(&services.ValidateService{}),
			application.NewService(&services.CheckPatternsService{}),
		},

		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},

		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "ScheduleGate",
		Width:            1400,
		Height:           900,
		MinWidth:         1024,
		MinHeight:        680,
		BackgroundColour: application.NewRGBA(8, 12, 20, 255),
		InitialPosition:  application.WindowCentered,
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
