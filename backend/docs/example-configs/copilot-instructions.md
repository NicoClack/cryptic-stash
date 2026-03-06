## Cryptic Stash
A web app for securely storing 2 factor recovery codes. Users can upload a file, which gets encrypted using a strong key derived from their password using Argon2ID, to create a stash. In the event the user loses access to their accounts, they can log in with the password and download the file after a waiting period. If an attacker tries to do the same, the user is notified and can block the attempt before it's allowed.

The project is a monorepo with a frontend folder, which uses SvelteKit and a backend which is written in Go.

# Instructions
- Prefer tabs to spaces, particularly with Go code, avoid adding space indentation in addition to the existing tabs on that line. This is to make diffs easier to read, but don't try to fix formatting too much, I'll run the formatter on save
- Before completing a request, ensure your changes didn't introduce any immediate syntax or linting errors. Use the language server for this initial check instead of the CLI. You can skip this step if your change isn't fully complete yet and you need further input