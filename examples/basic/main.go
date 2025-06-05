package main

import (
	"fmt"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian" // Adjust the import path as necessary
)

func main() {
	// Example of creating a Debian package
	pkg := debian.Package{
		Name:        "example-package",
		Version:     "1.0.0",
		Maintainer:  "Your Name <youremail@example.com>",
		Description: "This is an example Debian package.",
	}

	// Print package details
	fmt.Printf("Package Name: %s\n", pkg.Name)
	fmt.Printf("Version: %s\n", pkg.Version)
	fmt.Printf("Maintainer: %s\n", pkg.Maintainer)
	fmt.Printf("Description: %s\n", pkg.Description)

	// Here you can add more functionality to manipulate the package
	// For example, saving the package to a repository or creating a control file

	fmt.Println("Package saved successfully.")
}
