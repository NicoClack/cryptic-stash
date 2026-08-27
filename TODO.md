# TODO

- Create development Docker Compose setup
-   - Switch to Deno and limit postinstall scripts (locally and in CI)
- Update deprecated linters
- Check passkey implementation against https://developers.yubico.com/WebAuthn/WebAuthn_Developer_Guide/ . Recommendations:
-   - Consistently limit ceremonies to single use? e.g when there are unexpected errors
-   - Fix race conditions when using a WebAuthn session or creating one (it's currently created before the tx commits). Maybe store them in the database instead?
-   - Auto name passkeys based on AAGUID
- Add logout button and username in the top right
- Create AccountAlert system:
-   - Should be viewable when you log in
-   - Include logins, passkey changes, stash downloads etc
-   - Also create one when passkey is cloned, but don't demote for now
-   - Admin should be CC'd for some levels
- Replace adminCode
-   - Store admin passkeys in the database for easier management and to avoid maintaining two paths for the auth system
-   - Completing env setup should generate a 32 byte base64 encoded setup code. The generated environment variables include a `HASHED_ADMIN_SETUP_TOKEN` and `SETUP_MODE="admin-passkey"` (from `"env"`).
-   - When the server restarts, the page calls a setup endpoint with the token (saved in session storage) to register the first passkey
-   - The admin sets `SETUP_MODE` to `"completed"` and is required to remove the `HASHED_ADMIN_SETUP_TOKEN` env var
-   - The usual endpoints register and the admin is prompted to log in and then modify their passkeys as they want (I guess just track that state client side?)
-   - The admin messenger setup can remain unchanged
-   - Admin should still have sudo and non-sudo sessions, sudo sessions should get demoted after 5 minutes. Synced passkeys should be fine for non sensitive actions like reading logs, it also means the sessions can be shorter than they'd have to be for hardware keys
-   - ^ Need to implement sidevation for this to be secure though and enforce that the elevation passkey is sudo, rather than allowing sudo then non-sudo
-   - Remove dual group support? Does the threat model actually make sense? It only kind of works for regular users because the site is rarely accessed, but surely the admin shouldn't have less security than users?
- Enable WAL and update SQLite, there was a recent bug with it that could corrupt databases
- Enable SQLite's secure_delete pragma
- Replace entity struct arguments to services with IDs? Like passkeyManagement.go does
- Rework stash system
-   - Users should be able to manage their own stashes but maybe can't download them while logged in for now
-   - Remove the service-level encryption
-   - Use standard OPAQUE with all hashing on the client
-   -   - Makes things much simpler and less risky
-   -   - 256MB RAM and 1.5 seconds of compute is plenty secure, see 1Password's password strength research
-   -   - Might need to be increased in 5 years time but it would be a good idea to add a system for that anyway
-   -   - Makes the server practical to deploy on a small VM rather than just platforms like Railway
-   -   - Removes the main DDoS method, the only other likely one is SQLite locks
-   -   - Create hashing benchmark as the first env setup step. Remove old Go one
-   - Allow user to configure waiting period
- Replace non-standard Authorization headers with Bearer scheme
- Implement per user rate limits for guessing passwords, notify user once there's been enough. Global rate limit for all authentication done by an IP should be fine for passkey endpoints
- Remove old endpoints
- Use actors more consistently, should the invite functions enforce ownership? Currently the actor is only passed to the service, which uses it for logging
- Pass explicit dependencies to keyvalue, tempkeyvalue and ratelimiting packages rather than \*common.App
- Remove admin auth code logic
- Remove 2FA actions
- Prevent disabling main email messenger
- Allow general API and static asset rate limits to be set independently
-   - Maybe 180 requests per 2 minutes for API? 1.5 req/s
-   - Maybe 400 requests per 2 minutes for static assets
-   - If using a WAF, it's probably better to let it handle these, to minimise in-memory locks
- Block IPs who get passwords wrong too often. Use exponential backoff
- Send message when a stash password is correctly entered while it's locked
- Move package logging to the service layer, return result structs with the information needed to determine what to log
- Add password strength requirements, maybe using https://github.com/dropbox/zxcvbn or https://github.com/zxcvbn-ts/zxcvbn ? Use matcher-pwned
-   - Could a cost estimate be displayed instead? Similar to the experiments 1Password did
- Add /.well-known/change-password redirect
- Add /.well-known/security.txt redirect to GitHub
- Add /.well-known/passkey-endpoints so password managers know which pages to send the user to in order to configure passkeys
- Test security using standard tools, maybe make custom scripts too?
- Crash signals don't seem to show up in Railway. Is it because of the restart policy? Is the email only sent if the max is exceeded?
-   - Looks like it. Maybe recommend using Railway's webhooks in Discord or Slack?
-   - Or configure max restarts to zero, but then it will require a manual restart. Might be worth it depending on threat model though
-   - Recommend in the README to create a custom Railway notification for the project when a deployment is restarted?
- Limit the number of download sessions that can be created by a single IP to prevent 2FA fatigue attacks
-   - Maybe 3 per IP
-   - Have a per user limit as well of maybe 5. Cryptic Stash isn't designed to help you recover a compromised account, you're out of luck if an attacker has the correct password and locks you out from recovery
-   - Would also prevent denial of wallet attacks since each successful download session sends a message
- Display warning when signing into account if there's an active download session and prompt user to secure the stash
- Use "Cache-Control": "no-store" on sensitive endpoints?
- Use HSTS in Production
- Improve frontend security: CSP/HSTS/X-Frame-Options/X-Content-Type-Options/Referrer-Policy
- Generally improve the frontend
- Remove userID and publicMessage from logger, it's not worth the complexity and risks
-   - Maybe LoginAlerts should be used to display security messages when you log in?
- Fix elevation softlocks, see the StartElevation endpoint
- Can cancelling requests make views non-atomic if a view uses multiple transactions? Are there any security risks with this?
- Standardise returning errors and using gin.H vs the endpoint specific download struct. That struct applies defaults which the other 2 approaches don't, so it could leak information
- Implement Cloudflare Turnstile or reCAPTCHA. Turnstile is better for privacy so probably use that
-   - Probably makes the most sense to use for the stash download page. And the public signup page if I ever make that
- Avoid sending successful responses inside a transaction because it could fail while committing?
- Add limits on self-locking so a hacker can't lock you out forever
-   - Attempting to get an authorisation code when locked should send the unlock date
-   - Admins should be able to reset it so if there's an unauthorised login, the user can block with a self lock, the admin can reset them and then they can block again without waiting
- Create Cloudflare setup guide. Can I create IaC for the rate-limited endpoints?
- Use Discord API directly instead of discordgo
- Use final scratch image instead of Apline for running the backend, the build has no dependencies, so I don't even need commands like "ls"
- https://snyk.io/blog/go-security-cheatsheet-for-go-developers/
- Pass transactions explicitly
- Send warning message when a login uses the correct password but the account is locked
- Implement more messengers:
-   - ntfy.sh
-   - Webhooks?
-   - Matrix?
-   - Slack
- CSRF?
- Make non-email messengers act as echos, they should summarise the email and not provide any instructions. That way configuring more of them shouldn't really increase the phishing risk
- Create a PDF to explain the recovery and more-so the blocking process (they might not have the PDF in an recovery scenario).
- Move more logic out of endpoints
- CC admin (or all users?) when a user receives a login alert
- Review contexts. Possibly want to give them all a timeout, partly to make shutdowns more predictable
- Does log.Fatalf stop the shutdown logic running if the server crashes on startup?
- Send a message when messengers are changed, including the previous messengers. Include details about what login method was used to authorise it so if it was unauthorised, the user might know what went wrong when talking to the admin
- Cancel other download sessions when a download is completed
- Require both admin and users to click a link every 4 weeks (unless already locked) to confirm their contacts are working. If they don't click it, users will automatically lock and have to be unlocked by an admin. If the admin doesn't, all users will automatically lock
- Standardise frontend styling and headers
- Admin endpoints for troubleshooting:
-   - Dump database as sqlite file
-   - Cancel failed job
-   - Retry failed job
-   - Update job body
- Send regular messages to users and the admin
-   - Should have a clear message if nothing has happened, otherwise it displays totals for each type of message (e.g failed login) and all of the logs in chronological order
-   - Is it worth having general categories in logs (e.g login) like errors do?
-   - Occasionally have to click a link in it to verify that messenger is still working
-   -   - Should that link only be there when necessary?
- Audit use of time.sleep. Prefer time.After in a select so context cancellations can be respected
- Replace time calls with clock:
-   - Job engine checking which jobs are due
-   - Check sleep calls
- Recover panics in all of the service implementations and trigger a shutdown. They should recover once if it's a service like the database but otherwise remain shut down
- Prevent timing attacks from revealing if a user exists or not
-   - Create test with real endpoint, users in the test database and real hashing to see if an attacker could tell more than 80% of the time with 1000 samples. I guess disable the rate limiting though?
-   - The tests should have a singlethreaded and multithreaded variant to see if an increased server load reveals more information
-   - Can probably mitigate by waiting until a response time has been reached before sending the response. Maybe it would start at 1 second but it if it's ever exceeded, the new target would be a whole number of seconds. e.g 1.5 seconds of real processing time would result in a 2 second response time.
-   -   - How does this safely go down again? Going up isn't particularly safe either
-   - Admin endpoints don't need this security, as long as they fail early if unauthorised
- When the admin is locked, whether temporarily or permanently, errors should make the server enter some kind of lockdown state? Need to weigh up pros and cons
- Split alerts and other emails into 2 different addresses?
- govulncheck GitHub Action
- Standardise error handling on the frontend
- Use load functions on the frontend more consistently
- Page when invite link doesn't include an ID?
- Don't delete download sessions?
- Improved audit logging
- Reduce some of the duplication in test setup
- Delete accounts if they're locked for too long (GDPR)
-   - Lock accounts if the user doesn't respond to the regular messenger check.
-   - The email messenger probably shouldn't ever be disabled automatically? Should it be manually disableable?
- Periodically contact users and check they still know how to access their stash
-   - Don't include a link in these emails to reduce the phishing risk and to check they know the URL. Instead ask them to download their stash and click the dry run option
-   - Should have to tick a box to confirm they're using a guest browser profile
-   - Enter a dry run code from the email? Maybe that could reveal 3 random words that were included in the email that was sent when the stash was set up? Need to reduce the phishing risk somehow
-   - Then enter email and password. A confirmation email is sent.
- Delete old logs and other sources of PII periodically
- Improve frontend/local dev security:
-   - Use CSP to prevent fetches to other origins
-   - Use socket.dev to reduce chance of the frontend having malicious code? Create E2E test and see if any suspicious data is sent off
-   - Use npm-check-updates with a cooldown of a few days
- Use panic instead of Fatalf for startup errors?
- Restructure services so that implementations wrap errors defined in a more common package, e.g defined in twofactoractions/service.go. Messenger based implementation defined in twofactoractions/messengers/
- Review SQLite connection pool config
- Don't delete jobs on completion, instead periodically delete jobs older than 2 weeks or so. Could help with debugging
- Improve validation for messenger options
- If a credential is cloned, block sudo mode for it. Allow regular login to block downloads
- Should FinishRegisterPasskey demote existing sessions when a second group passkey is added for the first time?
- Improve service shutdowns, do any start the service first if it isn't already running like the scheduler used to do? The job engine might
- Is the commit system necessary for periodic tasks? Can't the number of calls just be counted and the time of the first call be recorded so it knows when next to run?

- When messengers are changed, send a message to all of the previous messengers
- Research step-security/harden-runner used by go-webauthn, could help against supply chain attacks
- Extend sessions while the user is logged in, but maybe not sudo ones?
- Prompt user to remove other sessions when they log in?
- Move from gin, its maintenance isn't great
- Add sudo recovery codes
-   - When a passkey clone is detected, demote the passkey to non-sudo and send a sudo recovery code
-   - When users log in, they can use the code to promote their active passkey to sudo
-   - Admin should also be able to send it in the event of partial lockouts, provided they verify the user's identity
- Research github.com/awnumar/memguard
- Refactor the logger
-   - Mostly to improve the self logging
- Change tests to use the test logger
- When using an authorisation code, look up general location using IP address and do some kind of fuzzy match on the user agent before allowing a download

# To watch

None

# Errors to investigate

- Clients cancelling requests causes internal errors?
  15:44:34 ERR middleware/logging.go:52 an internal server error occurred url=/api/v1/users/get-authorization-code/ method=POST error="db common [package] error: WithTx error: context canceled" statusCode=-1

- 00:07:52 ERR schedulers\delayFuncs.go:68 unable to create initial PeriodicTask object error="db common [package] error: WithTx error: start transaction error: database [general] error: other error: ent: starting a transaction: SQL logic error: cannot start a transaction within a transaction (1)" periodicTaskName=SEND_ACTIVE_SESSION_REMINDERS

- 11:42:03 ERR messengers/registry.go:328 failed to enqueue message send messengerType=discord_1 error="messengers [package] error: send error: enqueue job error: jobs [package] error: enqueue error: database [general] error: timeout [general] error: context deadline exceeded"
  11:42:03 ERR loggers/loggers.go:497 failed to message admin about an error error="db common [package] error: WithTx error: commit transaction error: database [general] error: other error: sql: transaction has already been committed or rolled back"
-   - Resolved? I made some fixes in this area

# To research

Can I zero sensitive memory on the frontend and backend?

Can I wake up a sleeping railway app by just having a separate cron service send an HTTP request over the internal network?

# Testing

- Expand download tests to cover self/admin locks
- Race condition fuzzer that spams a bunch of endpoints
-   - Would be run with the -race flag
-   - In particular, test that spamming get-authorization-code with the correct password then updating the password invalidates all of the codes generated using the old password
- Endpoints
-   - Do they cancel their work if a request times out?
- Invalid payloads for each endpoint, missing fields etc

# Test optimisations

- Don't save logs to database except for specific tests? Might make them a bit more realistic though
