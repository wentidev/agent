package utils

import (
	"flag"
)

var AppURL string
var AppToken string

func InitFlags() {
	flag.StringVar(&AppURL, "app-url", "https://app.wenti.dev", "The URL of the server")
	flag.StringVar(&AppToken, "app-token", "toto", "The Token for the server")
}
