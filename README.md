# Cryptic Stash

An E2E encrypted web app for securely storing 2 factor recovery codes in case you lose your devices or get locked out. You upload a file, which gets encrypted using your password using Argon2id and AES-256-GCM, in order to create a stash. In the event you lose access to your devices, you can log in with the password and download the file after a waiting period. If an attacker tries to do the same, you are notified and can block the attempt before it's allowed.

Technical features:

- Uses sudo and non-sudo passkeys for account management, maximising availability for defensive actions and security for sensitive actions
- Supports multiple messengers for redundancy and ensures users have been sufficiently notified before allowing a download
- Written in Go and uses SQLite for extremely low hosting costs on platforms like Railway
- Builds to a single portable binary via Docker (no CGo, frontend is embedded)
- Uses custom implementations of many services to increase security by reducing dependency count.

# Note

This project is still in a prerelease state and is not yet ready to use. I'm still working on a basic frontend and I'm expecting to have to make a few more breaking database schema changes. I expect to release a 1.0 version when I'm confident about hosting my own private 'production' instance, at which point the main branch will stabilise, I'll create a setup guide and I'll start using migrations.

# Planned Features

- Improved development security through Docker and Dev Containers
- Admins to use the same passkey auth system as regular users
- E2E encryption using OPAQUE key exchange
- Minimal CLI for encrypting/decrypting stashes, making the core security easier to audit and mitigating frontend supply chain attacks
- Finish frontend and setup rework, a lot of features are currently only implemented on the backend
- More polished frontend
- More admin controls
- Per user login rate limiting and alerting
- More messengers
- Further reduced dependencies
- Other plans and thoughts in the [TODO.md](./TODO.md)

# Licence

Licensed under GNU AGPL version 3, see [LICENSE.txt](./LICENSE.txt).
