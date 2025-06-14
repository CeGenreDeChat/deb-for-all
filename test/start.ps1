# Chemin vers l'interpréteur Python
$pythonPath = "python.exe"

# Chemin vers le fichier de test Robot Framework
$testFilePath = ".\suites\*.robot"
$outputdir = ".\results"

# Commande pour exécuter le test avec Robot Framework
$command = "$pythonPath -m robot --outputdir $outputdir $testFilePath"

# Exécuter la commande
Invoke-Expression $command
