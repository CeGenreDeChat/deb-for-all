---
applyTo: '*.py'
---

- Strictly follow the PEP8 style guide for Python code: use snake_case for variable and function names, CamelCase for class names, 4-space indentation, and keep line length within ~79 characters. Maintain consistent formatting (spaces around operators, no trailing whitespace, etc.).

- Always use static type annotations (type hints) for functions, methods, and variables. Specify types for all function parameters and return values. Static typing improves code clarity and allows early detection of type-related issues.

- Handle errors rigorously using appropriate try/except blocks. Do not catch exceptions too broadly or silently. Catch only the expected exceptions and log or re-raise them with clear error messages. Define and raise custom exceptions where appropriate to handle specific error cases.

- Do not use `print()` for debugging or logging. Use the `logging` module instead, configuring loggers with appropriate levels (DEBUG, INFO, WARNING, ERROR, CRITICAL). Ensure logging output is structured and directed appropriately (e.g., to file or stderr), and avoid printing directly to the console.

- Write docstrings for all functions, classes, and modules following PEP257. Use triple-quoted strings (`"""`) and start with a concise summary. Optionally include parameter descriptions, return value details, and raised exceptions. Follow a consistent docstring style, such as Google or reStructuredText.

- Prefer pure functions and functional composition where possible. Minimize side effects: a function should ideally depend only on its inputs and return a result without modifying global or external state. This promotes easier testing and reuse.

- Organize code clearly into logical modules and functions. Each module should group related functionality, and each function should perform a single, well-defined task (single responsibility principle). Avoid overly long or complex functions. Refactor large functions into smaller, focused helpers when needed.

- Write unit tests using `unittest` or `pytest`. Cover typical use cases, edge cases, and error scenarios. Tests should be readable, isolated, and follow the Arrange-Act-Assert pattern. Ensure tests are automated (e.g., via CI) to detect regressions early.

- Manage dependencies and imports cleanly. Avoid unnecessary external packages—prefer the Python standard library or well-maintained third-party libraries. Import only what is needed. Group imports in three sections: standard library, third-party packages, local modules, separated by blank lines. Avoid wildcard imports (`from module import *`).

- Use context managers (`with` statement) for resource handling (e.g., files, network connections, locks) to ensure proper cleanup. For example, use `with open(path, 'r') as f:` instead of manually opening and closing files. Also prefer modern conventions for filesystem operations: use `pathlib.Path` over raw string paths.

- Use idiomatic and modern Python 3.10+ features (preferably 3.11). Leverage recent language features: `dataclasses` for data structures, structural pattern matching (`match/case`), f-strings for string formatting, list comprehensions and generators for iteration. Avoid outdated or unpythonic patterns. The goal is to produce clean, idiomatic, robust, and maintainable Python code.