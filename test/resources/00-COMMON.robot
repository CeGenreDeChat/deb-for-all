*** Settings ***
Documentation     Keywords communs pour tous les tests de deb-for-all
...               Ce fichier contient les mots-clés réutilisables pour l'initialisation
...               et la configuration de l'environnement de test
Library           Process
Library           OperatingSystem
Library           Collections

*** Variables ***
${BINARY_NAME}          deb-for-all.exe

*** Keywords ***
Initialize Test Environment
    [Documentation]    Vérifie que le binaire deb-for-all.exe est présent et exécutable
    ...                Ce keyword DOIT être appelé avant tous les tests.
    [Tags]             prerequisite    initialization

    Log    Initialisation de l'environnement de test pour deb-for-all

    # Définir le chemin vers le binaire
    ${BINARY_PATH}=    Set Variable    ${CURDIR}${/}..${/}..${/}bin${/}${BINARY_NAME}
    Set Global Variable    ${BINARY_PATH}

    # Vérifier que le fichier binaire existe
    File Should Exist    ${BINARY_PATH}
    ...    msg=Le binaire ${BINARY_NAME} n'existe pas au chemin: ${BINARY_PATH}

    # Vérifier que le binaire est exécutable en testant la commande --help
    ${result}=    Run Process    ${BINARY_PATH}    --help

    # Le binaire DOIT retourner un code de sortie 0 pour --help
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=Le binaire n'est pas exécutable ou ne répond pas correctement à --help

    # Vérifier que la sortie contient les mots-clés attendus
    Should Contain    ${result.stdout}    Usage
    ...    msg=La sortie d'aide ne contient pas 'Usage'
    Should Contain    ${result.stdout}    download
    ...    msg=La commande 'download' n'est pas disponible dans le binaire
    Should Contain    ${result.stdout}    download-source
    ...    msg=La commande 'download-sorce' n'est pas disponible dans le binaire
    Should Contain    ${result.stdout}    mirror
    ...    msg=La commande 'mirror' n'est pas disponible dans le binaire

    Log    ✅ Binaire vérifié et opérationnel: ${BINARY_PATH}
    Log    Version et aide du binaire: ${result.stdout}

    # Retourner le chemin du binaire pour utilisation ultérieure
    RETURN    ${BINARY_PATH}

Create Test Directory
    [Documentation]    Crée un répertoire de test temporaire et le nettoie si nécessaire
    [Arguments]    ${directory_name}

    ${test_directory}=    Set Variable    ${TEMPDIR}${/}${directory_name}

    # Nettoyer le répertoire s'il existe déjà
    Run Keyword And Ignore Error    Remove Directory    ${test_directory}    recursive=True

    # Créer le nouveau répertoire
    Create Directory    ${test_directory}

    Log    Répertoire de test créé: ${test_directory}
    RETURN    ${test_directory}

Cleanup Test Directory
    [Documentation]    Nettoie un répertoire de test après utilisation
    [Arguments]    ${directory_path}

    Run Keyword And Ignore Error    Remove Directory    ${directory_path}    recursive=True
    Log    Répertoire de test nettoyé: ${directory_path}

Should Contain Any
    [Documentation]    Vérifie qu'un texte contient au moins une des chaînes fournies
    [Arguments]    ${text}    @{substrings}

    FOR    ${substring}    IN    @{substrings}
        ${contains}=    Run Keyword And Return Status    Should Contain    ${text}    ${substring}
        Return From Keyword If    ${contains}
    END

    Fail    Le texte "${text}" ne contient aucune des chaînes: ${substrings}