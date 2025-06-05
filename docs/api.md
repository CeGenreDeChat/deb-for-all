# API Documentation

## Overview

This document provides an overview of the API for the Debian package management library. It describes the various types and functions that are available for managing Debian packages.

## Package Types

### Package

```go
type Package struct {
    Name        string
    Version     string
    Architecture string
    Maintainer  string
    Description string
}
```

### ControlFile

```go
type ControlFile struct {
    Package     string
    Version     string
    Maintainer  string
    Architecture string
    Description string
}
```

## Functions

### ManagePackages

```go
func ManagePackages(pkg Package) error
```
This function manages the installation, removal, or upgrade of a Debian package.

### ReadControlFile

```go
func ReadControlFile(path string) (ControlFile, error)
```
This function reads a Debian control file from the specified path and returns a ControlFile struct.

### WriteControlFile

```go
func WriteControlFile(path string, control ControlFile) error
```
This function writes a ControlFile struct to the specified path.

### SearchPackage

```go
func SearchPackage(name string) ([]Package, error)
```
This function searches for packages by name and returns a list of matching packages.

## Error Handling

The library defines custom error types to handle specific errors related to package management. These errors can be used to provide more context when an operation fails.

## Usage Example

```go
package main

import (
    "fmt"
    "your_project/pkg/debian"
)

func main() {
    pkg := debian.Package{Name: "example", Version: "1.0"}
    err := debian.ManagePackages(pkg)
    if err != nil {
        fmt.Println("Error managing package:", err)
    }
}
```

## Conclusion

This API provides a comprehensive set of functions and types for managing Debian packages. For more detailed usage and examples, please refer to the documentation in the `examples` directory.