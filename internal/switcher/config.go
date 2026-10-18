package switcher

import (
	"hypr-dock/pkg/ini"
	"log"

	"github.com/hashicorp/go-hclog"
)

type Config struct {
	WidthPercent    int  `def:"100" max:"100"`
	HeightPercent   int  `def:"60" max:"100"`
	FontSize        int  `def:"32"`
	PreviewWidth    int  `def:"300"`
	Padding         int  `def:"40"`
	Spacing         int  `def:"30"`
	ShowAllMonitors bool `def:"false"`
	CycleWorkspaces bool `def:"true"`
	MouseSelection  bool `def:"false"`
	IconSize        int  `def:"0"`
}

func GetDefaultConfig() Config {
	return Config{
		WidthPercent:    100,
		HeightPercent:   60,
		FontSize:        20,
		PreviewWidth:    400,
		Padding:         40,
		Spacing:         30,
		ShowAllMonitors: false,
		CycleWorkspaces: true,
		MouseSelection:  false,
		IconSize:        0,
	}
}

func LoadConfig(path string) Config {
	cfg := ini.New(path, hclog.Default())

	var conf Config
	_, err := cfg.ParseSection(&conf, "General")
	if err != nil {
		return GetDefaultConfig()
	}

	log.Println(conf.FontSize)

	return conf
}
