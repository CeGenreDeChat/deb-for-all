---
applyTo: 'test/suites/*.robot'
---
# Instructions for Writing Robot Framework Test Files

## File Structure

1. **Clear Sections**: Ensure each test file is divided into well-defined sections: `Settings`, `Variables`, `Test Cases`, and `Keywords`.

2. **Documentation**:
   - **Test Suite**: Include general documentation for each test suite in the `Settings` section.
   - **Test Cases**: Each test case should have detailed documentation describing its purpose.
   - **Keywords**: Document each custom keyword to explain its purpose and functionality.

## Best Practices

1. **Descriptive Names**:
   - Use clear and descriptive names for test suites, test cases, and keywords.

2. **Use of Tags**:
   - Apply tags to test cases to facilitate selective test execution.

3. **Variables**:
   - Define and use variables for reusable values.
   - Avoid hard-coded values in test cases.

4. **Reusable Keywords**:
   - Create reusable keywords for common actions.
   - Use keyword libraries to extend functionality.

5. **Test Data Management**:
   - Separate test data from test cases using resource files or variable sections.

6. **File Organization**:
   - Structure your test files logically and maintain a clear folder hierarchy.

7. **How to Write Tests**:
   - The '[Return]' setting is deprecated. Use the 'RETURN' statement instead.
   - Separate Parameters and Values with Spaces.