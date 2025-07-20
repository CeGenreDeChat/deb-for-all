#!/usr/bin/env python3
"""
Script de démarrage des tests Robot Framework pour le projet deb-for-all.

"""

import argparse
import logging
import subprocess
import sys
from pathlib import Path
from typing import List, Optional


def setup_logging(verbose: bool = False) -> None:
    """Configure le système de logging."""
    level = logging.DEBUG if verbose else logging.INFO
    logging.basicConfig(
        level=level,
        format='%(asctime)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(sys.stderr)
        ]
    )

def build_robot_command(
    test_pattern: str,
    output_dir: Path,
    extra_args: Optional[List[str]] = None
) -> List[str]:
    """
    Construit la commande Robot Framework.
    
    Args:
        test_pattern: Pattern des fichiers de test
        output_dir: Répertoire de sortie
        extra_args: Arguments supplémentaires pour Robot Framework
        
    Returns:
        Liste des arguments de la commande
    """
    command = [
        'robot',
        '--outputdir', str(output_dir)
    ]
    
    if extra_args:
        command.extend(extra_args)
    
    command.append(test_pattern)
    return command


def run_robot_tests(
    test_pattern: str = "suites/*.robot",
    output_dir: str = "results",
    verbose: bool = False,
    extra_args: Optional[List[str]] = None
) -> int:
    """
    Execute les tests Robot Framework.
    
    Args:
        test_pattern: Pattern des fichiers de test à exécuter
        output_dir: Répertoire de sortie pour les résultats
        verbose: Mode verbose pour plus de logs
        extra_args: Arguments supplémentaires pour Robot Framework
        
    Returns:
        Code de retour du processus Robot Framework
    """
    setup_logging(verbose)
    
    # Conversion des chemins relatifs en Path objects
    current_dir = Path(__file__).parent
    test_path = current_dir / test_pattern
    results_path = current_dir / output_dir
    
    # Création du répertoire de sortie
    results_path.mkdir(exist_ok=True)
    logging.info(f"Répertoire de sortie: {results_path}")
    
    try:
        # Construction de la commande
        command = build_robot_command(
            str(test_path),
            results_path,
            extra_args
        )
        
        logging.info(f"Exécution: {' '.join(command)}")
        
        # Exécution des tests
        result = subprocess.run(command, cwd=current_dir)
        
        if result.returncode == 0:
            logging.info("Tous les tests ont réussi ✓")
        else:
            logging.warning(f"Tests échoués avec le code: {result.returncode}")
            
        logging.info(f"Résultats disponibles dans: {results_path}")
        return result.returncode
        
    except Exception as e:
        logging.error(f"Erreur lors de l'exécution des tests: {e}")
        return 1


def main() -> None:
    """Point d'entrée principal du script."""
    parser = argparse.ArgumentParser(
        description="Démarre les tests Robot Framework pour deb-for-all",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemples d'utilisation:
  python start.py                              # Tests par défaut
  python start.py --verbose                    # Mode verbose
  python start.py --pattern "suites/01-*.robot"  # Tests spécifiques
  python start.py --output-dir custom_results  # Répertoire personnalisé
  python start.py -- --include smoke          # Arguments Robot Framework
        """
    )
    
    parser.add_argument(
        '--pattern',
        default='suites/*.robot',
        help='Pattern des fichiers de test (défaut: suites/*.robot)'
    )
    
    parser.add_argument(
        '--output-dir',
        default='results',
        help='Répertoire de sortie (défaut: results)'
    )
    
    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='Mode verbose'
    )
    
    # Arguments supplémentaires pour Robot Framework
    parser.add_argument(
        'robot_args',
        nargs='*',
        help='Arguments supplémentaires pour Robot Framework'
    )
    
    args = parser.parse_args()
    
    exit_code = run_robot_tests(
        test_pattern=args.pattern,
        output_dir=args.output_dir,
        verbose=args.verbose,
        extra_args=args.robot_args if args.robot_args else None
    )
    
    sys.exit(exit_code)


if __name__ == '__main__':
    main()
