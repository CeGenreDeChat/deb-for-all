# Definitions
## MUST
This word, or the terms "REQUIRED" or "SHALL", mean that the definition is an absolute requirement of the specification.
## MUST NOT
This phrase, or the phrase "SHALL NOT", mean that the definition is an absolute prohibition of the specification.
## SHOULD
This word, or the adjective "RECOMMENDED", mean that there may exist valid reasons in particular circumstances to ignore a particular item, but the full implications must be understood and carefully weighed before choosing a different course.
## SHOULD NOT
This phrase, or the phrase "NOT RECOMMENDED" mean that there may exist valid reasons in particular circumstances when the particular behavior is acceptable or even useful, but the full implications should be understood and the case carefully weighed before implementing any behavior described with this label.

# Commit rules for Conventional Commits

* Commits MUST be prefixed with a type, which consists of a noun, feat, fix, etc., followed by the OPTIONAL scope, OPTIONAL !, and REQUIRED terminal colon and space.
* The type feat MUST be used when a commit adds a new feature to your application or library.
* The type fix MUST be used when a commit represents a bug fix for your application.
* A scope MAY be provided after a type. A scope MUST consist of a noun describing a section of the codebase surrounded by parenthesis, e.g., fix(parser):
* A description MUST immediately follow the colon and space after the type/scope prefix. The description is a short summary of the code changes, e.g., fix: array parsing issue when multiple spaces were contained in string.
* A longer commit body MAY be provided after the short description, providing additional contextual information about the code changes. The body MUST begin one blank line after the description.
* A commit body is free-form and MAY consist of any number of newline separated paragraphs.
* One or more footers MAY be provided one blank line after the body. Each footer MUST consist of a word token, followed by either a :<space> or <space># separator, followed by a string value (this is inspired by the git trailer convention).
* A footer’s token MUST use - in place of whitespace characters, e.g., Acked-by (this helps differentiate the footer section from a multi-paragraph body). An exception is made for BREAKING CHANGE, which MAY also be used as a token.
* A footer’s value MAY contain spaces and newlines, and parsing MUST terminate when the next valid footer token/separator pair is observed.
* Breaking changes MUST be indicated in the type/scope prefix of a commit, or as an entry in the footer.
* If included as a footer, a breaking change MUST consist of the uppercase text BREAKING CHANGE, followed by a colon, space, and description, e.g., BREAKING CHANGE: environment variables now take precedence over config files.
* If included in the type/scope prefix, breaking changes MUST be indicated by a ! immediately before the :. If ! is used, BREAKING CHANGE: MAY be omitted from the footer section, and the commit description SHALL be used to describe the breaking change.
* Types other than feat and fix MAY be used in your commit messages, e.g., docs: update ref docs.
* The units of information that make up Conventional Commits MUST NOT be treated as case sensitive by implementors, with the exception of BREAKING CHANGE which MUST be uppercase.
* BREAKING-CHANGE MUST be synonymous with BREAKING CHANGE, when used as a token in a footer.
* Commits MUST be written in english.

# Version rules

* Software using Semantic Versioning MUST declare a public API. This API could be declared in the code itself or exist strictly in documentation. However it is done, it SHOULD be precise and comprehensive.
* A normal version number MUST take the form X.Y.Z where X, Y, and Z are non-negative integers, and MUST NOT contain leading zeroes. X is the major version, Y is the minor version, and Z is the patch version. Each element MUST increase numerically. For instance: 1.9.0 -> 1.10.0 -> 1.11.0.
* Once a versioned package has been released, the contents of that version MUST NOT be modified. Any modifications MUST be released as a new version.
* Major version zero (0.y.z) is for initial development. Anything MAY change at any time. The public API SHOULD NOT be considered stable.
* Version 1.0.0 defines the public API. The way in which the version number is incremented after this release is dependent on this public API and how it changes.
* Patch version Z (x.y.Z | x > 0) MUST be incremented if only backward compatible bug fixes are introduced. A bug fix is defined as an internal change that fixes incorrect behavior.
* Minor version Y (x.Y.z | x > 0) MUST be incremented if new, backward compatible functionality is introduced to the public API. It MUST be incremented if any public API functionality is marked as deprecated. It MAY be incremented if substantial new functionality or improvements are introduced within the private code. It MAY include patch level changes. Patch version MUST be reset to 0 when minor version is incremented.
* Major version X (X.y.z | X > 0) MUST be incremented if any backward incompatible changes are introduced to the public API. It MAY also include minor and patch level changes. Patch and minor versions MUST be reset to 0 when major version is incremented.
* A pre-release version MAY be denoted by appending a hyphen and a series of dot separated identifiers immediately following the patch version. Identifiers MUST comprise only ASCII alphanumerics and hyphens [0-9A-Za-z-]. Identifiers MUST NOT be empty. Numeric identifiers MUST NOT include leading zeroes. Pre-release versions have a lower precedence than the associated normal version. A pre-release version indicates that the version is unstable and might not satisfy the intended compatibility requirements as denoted by its associated normal version. Examples: 1.0.0-alpha, 1.0.0-alpha.1, 1.0.0-0.3.7, 1.0.0-x.7.z.92, 1.0.0-x-y-z.--.
* Build metadata MAY be denoted by appending a plus sign and a series of dot separated identifiers immediately following the patch or pre-release version. Identifiers MUST comprise only ASCII alphanumerics and hyphens [0-9A-Za-z-]. Identifiers MUST NOT be empty. Build metadata MUST be ignored when determining version precedence. Thus two versions that differ only in the build metadata, have the same precedence. Examples: 1.0.0-alpha+001, 1.0.0+20130313144700, 1.0.0-beta+exp.sha.5114f85, 1.0.0+21AF26D3----117B344092BD.

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

# Project Goal
The aim of this project is to create a library for managing Debian packages within another Go project, and also to provide a binary for doing the same thing.
Do not create unitary tests.
For the example, juste use control, downloader, package and repository folder (with one main.go inside).