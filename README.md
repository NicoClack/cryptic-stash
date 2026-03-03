# Cryptic Stash
A web app for securely storing 2 factor recovery codes. Users can upload a file, which gets encrypted using a strong key derived from their password using Argon2ID, to create a stash. In the event the user loses access to their devices, they can log in with the password and download the file after a waiting period. If an attacker tries to do the same, the user is notified and can block the attempt before it's allowed.

Technical features:
- Written in Go and uses SQLite for extremely low hosting costs on platforms like Railway
- Uses custom implementations of many services to increase security by reducing dependency count
- Builds to a single portable binary via Docker (no CGo, frontend is embedded)
- Supports multiple messengers for redundancy and ensures users have been sufficiently notified before allowing a download

# Note
This project is still in a prerelease state and is not yet ready to use. I'm still working on a basic frontend and I'm expecting to have to make a few more breaking database schema changes. I expect to release a 1.0 version when I'm confident about hosting my own private 'production' instance, at which point the main branch will stabilise, I'll create a setup guide and I'll start using migrations.