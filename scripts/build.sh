#!/bin/bash

# Ce script automatise le processus de construction de l'application.

# Définir le nom du binaire
BINARY_NAME="deb-for-all"

# Construire le binaire
go build -o $BINARY_NAME ./cmd/deb-for-all/main.go

# Vérifier si la construction a réussi
if [ $? -eq 0 ]; then
    echo "Construction réussie : $BINARY_NAME"
else
    echo "Erreur lors de la construction."
    exit 1
fi

# Optionnel : exécuter les tests
go test ./... 

# Optionnel : nettoyer les fichiers temporaires
# rm -f $BINARY_NAME

# Fin du script