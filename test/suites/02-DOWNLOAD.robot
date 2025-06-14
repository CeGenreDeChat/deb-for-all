*** Settings ***
Documentation     Tests de la commande download du binaire deb-for-all
...               Cette suite valide le téléchargement de paquets binaires Debian
...               avec différentes options et configurations
Resource          ../resources/00-COMMON.robot

Suite Setup       Initialize Test Environment

Suite Teardown    Log    Tests de download terminés

*** Variables ***
# Paquets de test légers et fiables
${TEST_PACKAGE}         hello
${TEST_VERSION}         2.10-3
${TEST_ARCHITECTURE}    amd64
${SMALL_PACKAGE}        adduser
${SMALL_VERSION}        3.129ubuntu2
${TEST_TIMEOUT}         120s

*** Test Cases ***
Test Download Package Valid
    [Documentation]    Test du téléchargement d'un paquet valide
    [Tags]             download    valid    basic

    # Créer un répertoire de test pour le téléchargement
    ${download_dir}=    Create Test Directory    download-valid-test

    # Exécuter le téléchargement
    ${result}=    Run Process    ${BINARY_PATH}  -command  download
    ...    -package  ${TEST_PACKAGE}
    ...    -version  ${TEST_VERSION}
    ...    -dest  ${download_dir}

    # Vérifier que le téléchargement s'est bien déroulé
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=Échec du téléchargement: ${result.stderr}

    Log    Sortie du téléchargement: ${result.stdout}

    # Vérifier que le fichier .deb a été téléchargé
    ${expected_filename}=    Set Variable    ${TEST_PACKAGE}_${TEST_VERSION}_${TEST_ARCHITECTURE}.deb
    ${downloaded_file}=    Set Variable    ${download_dir}${/}${expected_filename}

    File Should Exist    ${downloaded_file}
    ...    msg=Le fichier ${expected_filename} n'a pas été téléchargé

    # Vérifier que le fichier n'est pas vide
    ${file_size}=    Get File Size  ${downloaded_file}
    Should Be True    ${file_size} > 0
    ...    msg=Le fichier téléchargé est vide

    Log    Fichier téléchargé avec succès: ${downloaded_file} (${file_size} bytes)

    [Teardown]    Cleanup Test Directory    ${download_dir}

Test Download Package Without Version
    [Documentation]    Test du téléchargement d'un paquet sans spécifier de version
    [Tags]             download    auto-version

    ${download_dir}=    Create Test Directory    download-auto-version

    # Télécharger sans spécifier de version (doit prendre la dernière)
    ${result}=    Run Process    ${BINARY_PATH}    -command    download
    ...    -package  ${SMALL_PACKAGE}
    ...    -dest  ${download_dir}
    ...    timeout=${TEST_TIMEOUT}

    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=Échec du téléchargement sans version: ${result.stderr}

    Log    Téléchargement auto-version: ${result.stdout}

    # Vérifier qu'un fichier .deb a été téléchargé
    @{deb_files}=    List Files In Directory    ${download_dir}    *.deb
    ${file_count}=    Get Length    ${deb_files}
    Should Be True    ${file_count} > 0
    ...    msg=Aucun fichier .deb trouvé dans ${download_dir}

    Log    Fichiers téléchargés: @{deb_files}

    [Teardown]    Cleanup Test Directory    ${download_dir}

Test Download Invalid Package
    [Documentation]    Test du téléchargement d'un paquet inexistant
    [Tags]             download    error-handling    negative

    ${download_dir}=    Create Test Directory    download-invalid-test

    # Tenter de télécharger un paquet inexistant
    ${result}=    Run Process    ${BINARY_PATH}  -command  download
    ...    -package  package-inexistant-test-123
    ...    -version  999.999-999
    ...    -dest  ${download_dir}
    ...    timeout=60s

    # Le téléchargement DOIT échouer
    Should Not Be Equal As Integers    ${result.rc}    0
    ...    msg=Le téléchargement aurait dû échouer pour un paquet inexistant

    Log    Erreur attendue: ${result.stderr}

    # Vérifier qu'aucun fichier .deb n'a été créé
    @{deb_files}=    List Files In Directory    ${download_dir}    *.deb
    ${file_count}=    Get Length    ${deb_files}
    Should Be Equal As Integers    ${file_count}    0
    ...    msg=Aucun fichier ne devrait être créé pour un paquet inexistant

    [Teardown]    Cleanup Test Directory    ${download_dir}

Test Download Missing Parameters
    [Documentation]    Test de validation des paramètres manquants
    [Tags]             download    validation    error-handling

    # Test sans nom de paquet
    ${result}=    Run Process    ${BINARY_PATH}  -command  download
    ...    -version  ${TEST_VERSION}

    Should Not Be Equal As Integers    ${result.rc}    0
    ...    msg=La commande aurait dû échouer sans nom de paquet

    Should Contain Any    ${result.stderr}    le nom du paquet est requis
    ...    msg=Message d'erreur attendu pour paquet manquant

    Log    Validation paramètres - Erreur paquet manquant: ${result.stderr}

Test Download Help
    [Documentation]    Test de l'aide de la commande download
    [Tags]             download    help    information

    ${result}=    Run Process    ${BINARY_PATH}    -command    download    -help

    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=La commande d'aide devrait réussir

    # Vérifier que l'aide contient les informations attendues
    Should Contain    ${result.stdout}    download
    Should Contain    ${result.stdout}    package
    Should Contain Any    ${result.stdout}    version    dest    destination

    Log    Aide de la commande download: ${result.stdout}

Test Download To Custom Directory
    [Documentation]    Test du téléchargement vers un répertoire personnalisé
    [Tags]             download    custom-directory

    ${custom_dir}=    Create Test Directory    custom-download-location
    ${subdir}=    Set Variable    ${custom_dir}${/}packages${/}debian
    Create Directory    ${subdir}

    # Télécharger vers le sous-répertoire personnalisé
    ${result}=    Run Process    ${BINARY_PATH}    -command    download
    ...    -package    ${TEST_PACKAGE}
    ...    -version    ${TEST_VERSION}
    ...    -dest    ${subdir}

    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=Échec du téléchargement vers répertoire personnalisé: ${result.stderr}

    # Vérifier que le fichier est dans le bon répertoire
    ${expected_filename}=    Set Variable    ${TEST_PACKAGE}_${TEST_VERSION}_${TEST_ARCHITECTURE}.deb
    ${downloaded_file}=    Set Variable    ${subdir}${/}${expected_filename}

    File Should Exist    ${downloaded_file}
    ...    msg=Le fichier n'a pas été téléchargé dans le répertoire personnalisé

    Log    Téléchargement réussi vers: ${downloaded_file}

    [Teardown]    Cleanup Test Directory    ${custom_dir}

Test Download Silent Mode
    [Documentation]    Test du téléchargement en mode silencieux
    [Tags]             download    silent    mode

    ${download_dir}=    Create Test Directory  download-silent-test

    # Télécharger en mode silencieux
    ${result}=    Run Process    ${BINARY_PATH}  -command    download
    ...    -package  ${TEST_PACKAGE}
    ...    -version  ${TEST_VERSION}
    ...    -dest  ${download_dir}
    ...    -silent

    Should Be Equal As Integers  ${result.rc}  0
    ...    msg=Échec du téléchargement en mode silencieux: ${result.stderr}

    # En mode silencieux, la sortie devrait être minimale
    ${stdout_length}=    Get Length  ${result.stdout}
    Log    Sortie en mode silencieux (${stdout_length} caractères): ${result.stdout}

    # Vérifier que le fichier a bien été téléchargé malgré le mode silencieux
    ${expected_filename}=    Set Variable    ${TEST_PACKAGE}_${TEST_VERSION}_${TEST_ARCHITECTURE}.deb
    File Should Exist    ${download_dir}${/}${expected_filename}
    ...    msg=Le fichier devrait être téléchargé même en mode silencieux

    [Teardown]    Cleanup Test Directory    ${download_dir}
