package main

import "github.com/spf13/cobra"

func initCommands() {
	// Initialiser la commande racine
	rootCmd = &cobra.Command{
		Use:   "deb-for-all",
		Short: "Debian package management tool",
	}

	// Flags globaux
	rootCmd.PersistentFlags().StringVarP(&config.DestDir, "dest", "d", "./downloads", localize("flag.dest"))
	rootCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, localize("flag.verbose"))
	rootCmd.PersistentFlags().StringVar(&config.CacheDir, "cache", "./cache", localize("flag.cache"))
	rootCmd.PersistentFlags().StringVar(&config.Keyrings, "keyring", "", localize("flag.keyring"))
	rootCmd.PersistentFlags().BoolVar(&config.NoGPGVerify, "no-gpg-verify", false, localize("flag.no_gpg_verify"))

	// Commande `download`
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: localize("command.download"),
		Run: func(cmd *cobra.Command, args []string) {
			config.Command = "download"
		},
	}
	downloadCmd.Flags().StringVarP(&config.PackageName, "package", "p", "", localize("flag.package"))
	downloadCmd.Flags().StringVar(&config.Version, "version", "", localize("flag.version"))
	downloadCmd.Flags().BoolVarP(&config.Silent, "silent", "s", false, localize("flag.silent"))
	downloadCmd.MarkFlagRequired("package")
	rootCmd.AddCommand(downloadCmd)

	// Commande `download-source`
	downloadSourceCmd := &cobra.Command{
		Use:   "download-source",
		Short: localize("command.download_source"),
		Run: func(cmd *cobra.Command, args []string) {
			config.Command = "download-source"
		},
	}
	downloadSourceCmd.Flags().StringVarP(&config.PackageName, "package", "p", "", localize("flag.package"))
	downloadSourceCmd.Flags().StringVar(&config.Version, "version", "", localize("flag.version"))
	downloadSourceCmd.Flags().BoolVar(&config.OrigOnly, "orig-only", false, localize("flag.orig_only"))
	downloadSourceCmd.Flags().BoolVarP(&config.Silent, "silent", "s", false, localize("flag.silent"))
	downloadSourceCmd.MarkFlagRequired("package")
	rootCmd.AddCommand(downloadSourceCmd)

	// Commande `update`
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: localize("command.update"),
		Run: func(cmd *cobra.Command, args []string) {
			config.Command = "update"
		},
	}
	updateCmd.Flags().StringVar(&config.BaseURL, "url", "http://deb.debian.org/debian", localize("flag.url"))
	updateCmd.Flags().StringVar(&config.Suites, "suites", "bookworm", localize("flag.suites"))
	updateCmd.Flags().StringVar(&config.Components, "components", "main", localize("flag.components"))
	updateCmd.Flags().StringVar(&config.Architectures, "architectures", "amd64", localize("flag.architectures"))
	rootCmd.AddCommand(updateCmd)

	// Commande `mirror`
	mirrorCmd := &cobra.Command{
		Use:   "mirror",
		Short: localize("command.mirror"),
		Run: func(cmd *cobra.Command, args []string) {
			config.Command = "mirror"
		},
	}
	mirrorCmd.Flags().StringVarP(&config.BaseURL, "url", "u", "http://deb.debian.org/debian", localize("flag.url"))
	mirrorCmd.Flags().StringVar(&config.Suites, "suites", "bookworm", localize("flag.suites"))
	mirrorCmd.Flags().StringVar(&config.Components, "components", "main", localize("flag.components"))
	mirrorCmd.Flags().StringVar(&config.Architectures, "architectures", "amd64", localize("flag.architectures"))
	mirrorCmd.Flags().BoolVar(&config.DownloadPkgs, "download-packages", false, localize("flag.download_packages"))
	rootCmd.AddCommand(mirrorCmd)
}
