package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/CeGenreDeChat/deb-for-all/cmd/deb-for-all/commands"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localesFS embed.FS

// Config globale pour stocker les arguments
type Config struct {
	Command       string
	PackageName   string
	Version       string
	DestDir       string
	CacheDir      string
	OrigOnly      bool
	Silent        bool
	BaseURL       string
	Suites        string
	Components    string
	Architectures string
	DownloadPkgs  bool
	Verbose       bool
}

var (
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
	config    Config
	rootCmd   *cobra.Command
)

func initI18n() {
	// Initialiser le bundle de traductions
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Charger les fichiers de traduction embarqués
	bundle.LoadMessageFileFS(localesFS, "locales/en.toml")
	bundle.LoadMessageFileFS(localesFS, "locales/fr.toml")

	// Détecter la langue (par défaut: anglais)
	lang := os.Getenv("DEB_FOR_ALL_LANG")
	if lang == "" {
		lang = "en"
	}
	localizer = i18n.NewLocalizer(bundle, lang)
}

func localize(key string) string {
	msg, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: key})
	return msg
}

func run() error {
	switch strings.ToLower(config.Command) {
	case "download":
		return commands.DownloadBinaryPackage(config.PackageName, config.Version, config.DestDir, config.Silent, localizer)
	case "download-source":
		return commands.DownloadSourcePackage(config.PackageName, config.Version, config.DestDir, config.OrigOnly, config.Silent, localizer)
	case "mirror":
		return commands.CreateMirror(config.BaseURL, config.Suites, config.Components, config.Architectures, config.DestDir, config.DownloadPkgs, config.Verbose, localizer)
	case "update":
		return commands.UpdateCache(config.BaseURL, config.Suites, config.Components, config.Architectures, config.CacheDir, config.Verbose, localizer)
	default:
		return errors.New(localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "error.unknown_command",
			TemplateData: map[string]interface{}{
				"Command": config.Command, // Passe la commande comme variable
			},
		}))
	}
}

func main() {
	// Initialiser i18n en premier
	initI18n()

	// Initialiser les commandes Cobra
	initCommands()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Appeler run() après l'exécution de Cobra
	if config.Command != "" {
		if err := run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
