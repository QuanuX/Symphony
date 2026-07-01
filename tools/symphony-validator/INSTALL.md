# Symphony Validator Installation

**Candidate planning draft only. Not canonical. Not imported.**

## Installation Status
Not yet implemented.

## Future Installation Intent
The validator will be an individually installable native tool built with a portable C++ build model (CMake or Make).

## Development Posture
Invoked as a standalone tool during local development and preflight checks.

## Production Posture
Invoked by CI systems and the administrative spine for structural and doctrinal verification.

## Non-requirements
This seed does not provide install commands.
This seed does not provide build commands.
This seed does not authorize validator implementation.

## Python Doctrine
Python must not be required for remote native hot-path execution or the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.
Choosing C++ for the validator does not ban optional isolated Python habitats.
It prevents Python from becoming required validator infrastructure for the administrative spine.

## Non-authorization Statement
This candidate does not authorize canonical repository mutation, C++ validator implementation, or executable schema generation.
