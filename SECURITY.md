# Security policy

## Supported versions

Only the latest release of ppm receives security fixes. Update with
`ppm self update` or by re-running the install script.

## Reporting a vulnerability

Please do **not** open a public issue for security problems.

Report it privately through GitHub:
**Security → Report a vulnerability** on
https://github.com/rahulcodepython/PPM/security/advisories/new

Include the ppm version (`ppm version`), your OS, and steps to reproduce.
You should get a reply within 7 days. Once a fix is released the advisory
is published and you are credited unless you ask otherwise.

## Scope

ppm runs pip/uv, Python and the scripts in your `ppm.yml`. Packages you
install and scripts you define run with your user's permissions; that is
expected behaviour, not a vulnerability. In scope are issues in ppm itself,
for example:

- `ppm self update`, `ppm self install` or the install scripts accepting a tampered binary
- ppm executing commands that are not in `ppm.yml` or on the command line
- ppm writing files outside the project directory
