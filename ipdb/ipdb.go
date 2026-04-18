package ipdb

import (
	"log/slog"

	"resty.dev/v3"
)

func PullDatabase() (error, error) {
	client := resty.New().SetOutputDirectory("./")
	slog.Info("Downloading database...")
	_, err := client.R().SetSaveResponse(true).Get("https://gh.llkk.cc/https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb")
	if err != nil {
		slog.Error("Error downloading database", "error", err)
	}
	_, errv6 := client.R().SetSaveResponse(true).Get("https://gh.llkk.cc/https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v6.xdb")
	if errv6 != nil {
		slog.Error("Error downloading database", "error", errv6)
	}
	slog.Info("Download Successfully!")
	return err, errv6
}
