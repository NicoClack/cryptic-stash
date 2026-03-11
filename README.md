# Cryptic Stash
An inverse 2FA web app for securely storing 2 factor recovery codes in case you lose your devices or get locked out. You can upload a file, which gets encrypted using a strong key ultimately derived from your password using Argon2ID, to create a stash. In the event you lose access to your devices, you can log in with the password and download the file after a waiting period. If an attacker tries to do the same, you are notified and can block the attempt before it's allowed.

Technical features:
- Written in Go and uses SQLite for extremely low hosting costs on platforms like Railway
- Uses custom implementations of many services to increase security by reducing dependency count
- Builds to a single portable binary via Docker (no CGo, frontend is embedded)
- Supports multiple messengers for redundancy and ensures users have been sufficiently notified before allowing a download

# Note
This project is still in a prerelease state and is not yet ready to use. I'm still working on a basic frontend and I'm expecting to have to make a few more breaking database schema changes. I expect to release a 1.0 version when I'm confident about hosting my own private 'production' instance, at which point the main branch will stabilise, I'll create a setup guide and I'll start using migrations.