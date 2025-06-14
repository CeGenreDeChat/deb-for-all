*** Settings ***
Documentation     Cette fiche concerne le clonage d'un dépôt Debian

Resource          ..${/}resources${/}00-COMMON.robot

Library           Collections    # Importe la bibliothèque Collections, utile pour les listes et dictionnaires

Suite Setup       Initialize Test Environment

*** Variables ***
${REPOSITORY_URL}   http://deb.debian.org/debian
${SUITE}            bullseye
${COMPONENTS}       main
${ARCHITECTURE}     amd64

*** Test Cases ***
Test Clonage Métadonnées Uniquement
    [Documentation]    Test du clonage d'un dépôt Debian avec métadonnées uniquement
    [Tags]             mirror    metadata    basic

    # Créer un répertoire de test
    ${mirror_directory}=    Create Test Directory    debian-mirror-metadata

    # Exécuter le clonage avec métadonnées uniquement
    ${result}=    Run Process    ${BINARY_PATH}  -command  mirror
    ...    -url  ${REPOSITORY_URL}
    ...    -suites  ${SUITE}
    ...    -components  ${COMPONENTS}
    ...    -architectures  ${ARCHITECTURE}
    ...    -dest  ${mirror_directory}

    # Vérifier que le clonage s'est bien déroulé
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=Échec du clonage: ${result.stderr}

    Log    Sortie du clonage: ${result.stdout}

    # Valider la structure du dépôt cloné
    Verify Debian Dists Repository Structure    ${mirror_directory}    ${SUITE}    ${COMPONENTS}    ${ARCHITECTURE}

    [Teardown]    Cleanup Test Directory    ${mirror_directory}

Test Validation Commande Mirror
    [Documentation]    Test de validation des paramètres de la commande mirror
    [Tags]             validation    error-handling

    # Test avec URL invalide
    ${result}=    Run Process    ${BINARY_PATH}  -command  mirror
    ...    -url  http://invalid.repository.url
    ...    -suites  ${SUITE}
    ...    -components  ${COMPONENTS}
    ...    -architectures  ${ARCHITECTURE}
    ...    -dest  ${TEMPDIR}${/}invalid-test

    # Le clonage DOIT échouer avec une URL invalide
    Should Not Be Equal As Integers    ${result.rc}    0
    ...    msg=Le clonage aurait dû échouer avec une URL invalide

    Log    Erreur attendue: ${result.stderr}

Test Information Mirror
    [Documentation]    Test de récupération des informations du mirror
    [Tags]             information    status

    ${test_directory}=    Create Test Directory    debian-mirror-info

    # Exécuter la commande d'information
    ${result}=    Run Process    ${BINARY_PATH}    mirror
    ...    -url  ${REPOSITORY_URL}
    ...    -suites  ${SUITE}
    ...    -components  ${COMPONENTS}
    ...    -architectures  ${ARCHITECTURE}
    ...    -dest  ${test_directory}

    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=Échec de récupération des informations: ${result.stderr}

    # Vérifier que les informations attendues sont présentes
    Should Contain Any    ${result.stdout}    Mirror    Information    URL    Suites

    Log    Informations du mirror: ${result.stdout}

    [Teardown]    Cleanup Test Directory    ${test_directory}

*** Keywords ***
Verify Debian Dists Repository Structure
    [Documentation]    Vérifie qu'un répertoire contient la structure standard d'un dépôt Debian
    [Arguments]    ${repository_path}    ${suite}    ${component}    ${architecture}

    # Vérifier la structure de base
    Directory Should Exist    ${repository_path}${/}dists

    # Vérifier la structure spécifique à la suite
    Directory Should Exist    ${repository_path}${/}dists${/}${suite}
    File Should Exist    ${repository_path}${/}dists${/}${suite}${/}Release

    # Vérifier la structure du composant
    Directory Should Exist    ${repository_path}${/}dists${/}${suite}${/}${component}
    Directory Should Exist    ${repository_path}${/}dists${/}${suite}${/}${component}${/}binary-${architecture}

    # Vérifier que le fichier Packages existe (compressé ou non)
    ${packages_dir}=    Set Variable    ${repository_path}${/}dists${/}${suite}${/}${component}${/}binary-${architecture}

    ${packages_gz_exists}=    Run Keyword And Return Status
    ...    File Should Exist    ${packages_dir}${/}Packages.gz
    ${packages_exists}=    Run Keyword And Return Status
    ...    File Should Exist    ${packages_dir}${/}Packages

    Run Keyword If    not ${packages_gz_exists} and not ${packages_exists}
    ...    Fail    Aucun fichier Packages trouvé dans ${packages_dir}

    Log    ✅ Structure du dépôt Debian validée pour ${repository_path}

Verify Debian Pool Repository Structure
    [Documentation]    Vérifie qu'un répertoire contient la structure standard d'un dépôt Debian
    [Arguments]    ${repository_path}    ${suite}    ${component}    ${architecture}

    # Vérifier la structure de base
    Directory Should Exist    ${repository_path}${/}pool

    # Vérifier la structure du composant
    Directory Should Exist    ${repository_path}${/}pool${/}${component}

    Log    ✅ Structure du dépôt Debian validée pour ${repository_path}