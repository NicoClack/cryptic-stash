# Cryptic Stash
An inverted 2FA web app for securely storing 2 factor recovery codes in case you lose your devices or get locked out. You can upload a file, which gets encrypted using a strong key ultimately derived from your password using Argon2ID, to create a stash. In the event you lose access to your devices, you can log in with the password and download the file after a waiting period. If an attacker tries to do the same, you are notified and can block the attempt before it's allowed.

Technical features:
- Written in Go and uses SQLite for extremely low hosting costs on platforms like Railway
- Uses custom implementations of many services to increase security by reducing dependency count
- Builds to a single portable binary via Docker (no CGo, frontend is embedded)
- Supports multiple messengers for redundancy and ensures users have been sufficiently notified before allowing a download

# Planned Features
- E2E encryption with lower-memory hashing on the client and higher-memory hashing on the server (or client if using CLI)
- - Would replace the current encryption at rest system which requires the server to be trusted with the user's password and unencrypted stash
- - [Setup process diagram](https://docs.google.com/drawings/d/1j9TDj7PY13-t-lmEfODnJESU8F0CO0T38Qf0qzYkmuE/edit?usp=sharing)
- - [Semi trust download diagram](https://docs.google.com/drawings/d/1LrkucakN_f_NvjjX-dNF1Lol_9Minyr8F5_bS9xp8q8/edit?usp=sharing)
- - [Zero trust download digram](https://docs.google.com/drawings/d/1Wa0TlPGFmKDCo3EMr-2fTZeeeR019FGWj_h3s42QT7Y/edit?usp=sharing)
- Prevent user enumeration
- - User passwords should really be secure anyway but I want Cryptic Stash to be as secure as possible. Plus there are potential privacy/compliance implications of leaving it in
- - Will require using emails instead of usernames
- Finish frontend and setup rework, a lot of features are currently only implemented on the backend
- Give more control to users rather than having to rely on the admin
- - Cryptic Stash probably can't save you if your accounts are hacked, so as long as stashes can't be downloaded without the password (which doesn't have to be stored in your password manager), the inconvenience isn't worth it
- - Plus relying on the admin too much could increase social engineering attacks
- - It would also be harder for users to keep their messengers up-to-date
- Improved rate limiting and limits to concurrent hashing
- More messengers
- Further reduced dependencies
- I'm currently trying to decide on a licence, although it will most likely be GPL3 or AGPL3
- Other plans and thoughts in the [TODO.md](./TODO.md)

# Note
This project is still in a prerelease state and is not yet ready to use. I'm still working on a basic frontend and I'm expecting to have to make a few more breaking database schema changes. I expect to release a 1.0 version when I'm confident about hosting my own private 'production' instance, at which point the main branch will stabilise, I'll create a setup guide and I'll start using migrations.