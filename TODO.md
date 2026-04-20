# TODO

- Create account system with passkeys
- Remove admin auth code logic
- Add unique userID + publicName index to stashes
- Rework functions in core/users.go to take stashes instead
- Remove 2FA actions
- Prevent disabling main email messenger
- Replace env.STASH_ENCRYPTION_KEY with key derivation algorithm
- Move env encryption from the service? Stash content and filenames don't need to be encrypted by it because the encryption keys for them are encrypted with the env var
- Rate limit hashing
- - Limit number of concurrent hash requests to avoid using too much RAM
- - Block IPs who get passwords wrong too often. Use exponential backoff
- Use hash to store code in invite links rather than search param, that way it doesn't show up in logs
- Prevent username enumeration? Does user ID enumeration matter if the login endpoint only accepts usernames?
- - n8n just seems to mitigate with jitter, probably enough: https://github.com/n8n-io/n8n/pull/24553/changes
- - Reset emails should always succeed unless triggered by an admin (maybe only implement the latter for now)
- - Invites should specify the email and not allow it to be changed, like n8n
- - If using usernames on signup page, could mitigate by limiting failed uses of an invite link and requiring a unique credit card for public signups (if I ever implement that)
- - Can fake salts by hashing something deterministic like the email with a static pepper. If the pepper is leaked, I can just rotate it. If the database is leaked, the attacker has the stash records anyway. This approach means I don't need to store the random data I'm sending to ensure consistent results 
- Create account system for users to manage their stashes
- - Require FIDO2 passkeys/physical security keys
- - Allow optional 2FA FIDO2 with "userVerification" set to "discouraged" by default, allowing it to be set to "required". Intended for login via password manager + security key
- - Locking account permanently or temporarily should only require a single credential, either a first or a second factor. That way you can block attempts with just access to your password manager or a security key with someone else's device
- - User logins can be reset by the admin generating a link. Maybe it could also require a code sent to their email?
- - Admin logins can be reset by changing env vars
- Send message when a stash password is correctly entered while it's locked
- Implement some form of E2E encryption
- - My design is probably secure but would be best to stick to a standard system if possible. Although it looks like different password managers use different ones
- - I think OPAQUE has been mathematically proven but its Go implementation is too new.
- - - https://eprint.iacr.org/2018/163.pdf
- - - https://github.com/bytemare/opaque
- - - https://blog.cloudflare.com/opaque-oblivious-passwords/
- - - https://csrc.nist.gov/csrc/media/presentations/2024/crclub-2024-10-16/images-media/crypto-club-20241016--hugo--OPAQUE.pdf
- - Secure Remote Password isn't mathematically proven but has so far been secure enough for important services like 1Password to use (though they might be moving away from it). Might be easier to implement
- Add password strength requirements, maybe using https://github.com/dropbox/zxcvbn or https://github.com/zxcvbn-ts/zxcvbn ? Use matcher-pwned
- Add /.well-known/change-password redirect
- Add /.well-known/security.txt redirect to GitHub
- Add /.well-known/passkey-endpoints so password managers know which pages to send the user to in order to configure passkeys
- Test security using standard tools, maybe make custom scripts too?
- Crash signals don't seem to show up in Railway. Is it because of the restart policy? Is the email only sent if the max is exceeded?
- - Looks like it. Maybe recommend using Railway's webhooks in Discord or Slack?
- - Or configure max restarts to zero, but then it will require a manual restart. Might be worth it depending on threat model though
- - Recommend in the README to create a custom Railway notification for the project when a deployment is restarted?
- Limit the number of download sessions that can be created by a single IP to prevent 2FA fatigue attacks
- - Would also prevent denial of wallet attacks since each successful download session sends a message
- Allow updating stash contents/password
- Use "Cache-Control": "no-store" on sensitive endpoints?
- Disk usage keeps increasing. Maybe need to delete old job executions and logs? Implement the dump database endpoint so I can inspect
- Use HSTS in Production
- Improve frontend security: CSP/HSTS/X-Frame-Options/X-Content-Type-Options/Referrer-Policy
-   Improve frontend
-   Remove userID and publicMessage from logger, it's not worth the complexity and risks
- Encrypt stashes with an extra key to prevent offline attacks if database is leaked
-   Can cancelling requests make views non-atomic if a view uses multiple transactions? Are there any security risks with this?
-   Standardise returning errors and using gin.H vs the endpoint specific download struct. That struct applies defaults which the other 2 approaches don't, so it could leak information
-   Experiment using Cloudflare to prevent DDoS requests on the hashing endpoint. It's not a great idea to shift the hashing to the client due to WASM and different devices' RAM limitations. Can specifically limit that endpoint
- Implement Cloudflare Turnstile or reCAPTCHA. Turnstile is better for privacy so probably use that
-   Avoid sending successful responses inside a transaction because it could fail while committing?
-   Add limits on self-locking so a hacker can't lock you out forever
-   -   Attempting to get an authorisation code when locked should send the unlock date
-   Repeat password in sign up form
-   -   Admins should be able to reset it so if there's an unauthorised login, the user can block with a self lock, the admin can reset them and then they can block again without waiting
- Use Discord API directly instead of discordgo
- Use final scratch image instead of Apline for running the backend, the build has no dependencies, so I don't even need commands like "ls"
- https://snyk.io/blog/go-security-cheatsheet-for-go-developers/
- Remove clockwork and use Go's built-in mock time
- Pass transactions explicitly
- Send warning message when a login uses the correct password but the account is locked
- Implement more messengers:
- - Email (SendGrid)
- - ntfy.sh
- - Matrix?
- - Slack
- - Probably not SMS or Signal as they require renting a phone number
- - Not WhatsApp because their business API seems expensive
-   CSRF?
-   Move more logic out of endpoints
-   CC admin (or all users?) when a user receives a login alert
-   Review contexts. Possibly want to give them all a timeout, partly to make shutdowns more predictable
-   Does log.Fatalf stop the shutdown logic running if the server crashes on startup?
-   Require both admin and users to click a link every 4 weeks (unless already locked) to confirm their contacts are working. If they don't click it, users will automatically lock and have to be unlocked by an admin. If the admin doesn't, all users will automatically lock
- Standardise frontend styling and headers
-   Admin endpoints for troubleshooting:
-   -   Dump database as sqlite file
-   -   Cancel failed job
-   -   Retry failed job
-   -   Update job body
-   Send regular messages to users and the admin
-   -   Should have a clear message if nothing has happened, otherwise it displays totals for each type of message (e.g failed login) and all of the logs in chronological order
-   -   Is it worth having general categories in logs (e.g login) like errors do?
-   -   Occasionally have to click a link in it to verify that messenger is still working
-   -   -   Should that link only be there when necessary?
-   Audit use of time.sleep. Prefer time.After in a select so context cancellations can be respected
-   Recover panics in all of the service implementations and trigger a shutdown. They should recover once if it's a service like the database but otherwise remain shut down
-   Prevent timing attacks from revealing if a user exists or not
-   -   Create test with real endpoint, users in the test database and real hashing to see if an attacker could tell more than 80% of the time with 1000 samples. I guess disable the rate limiting though?
-   -   The tests should have a singlethreaded and multithreaded variant to see if an increased server load reveals more information
-   -   Can probably mitigate by waiting until a response time has been reached before sending the response. Maybe it would start at 1 second but it if it's ever exceeded, the new target would be a whole number of seconds. e.g 1.5 seconds of real processing time would result in a 2 second response time.
-   -   -   How does this safely go down again? Going up isn't particularly safe either
-   -   Admin endpoints don't need this security, as long as they fail early if unauthorised
-   When the admin is locked, whether temporarily or permanently, errors should make the server enter some kind of lockdown state? Need to weigh up pros and cons
- Split alerts and other emails into 2 different addresses
- govulncheck GitHub Action
- Standardise error handling on the frontend
- Use load functions on the frontend more consistently
- Page when invite link doesn't include an ID?
- Don't delete download sessions?
- Improved audit logging
- Reduce some of the duplication in test setup
- Delete accounts if they're locked for too long (GDPR)
- - Lock accounts if the user doesn't respond to the regular messenger check.
- - The email messenger probably shouldn't ever be disabled automatically? Should it be manually disableable?
- Delete old logs and other sources of PII periodically
- Improve frontend/local dev security:
- - Switch to Deno and limit postinstall scripts (locally and in CI)
- - Use CSP to prevent fetches to other origins
- - Use socket.dev to reduce chance of the frontend having malicious code? Create E2E test and see if any suspicious data is sent off
- - Use npm-check-updates with a cooldown of a few days

- Move from gin, its maintenance isn't great
- When messengers are changed, send a message to all of the previous messengers
- Allow user to increase waiting period, users could create a second account for a digital legacy. Although would that require some kind of split password system?
- Don't delete jobs on completion, instead periodically delete jobs older than 2 weeks or so. Could help with debugging
-   Rework endpoint system, maybe the endpoint functions could return an Endpoint struct with an array of handlers and some other things? Middleware should be defined there instead of in RegisterEndpoints
-   Create servicescommon so things can be split up better?
-   Job engine should support rate limiting for each API by each definition having an optional function to modify the database object.
    There could be a function to increase the due time based on the internal rate limit for the API. Probably not needed though
-   Refactor the logger
-   -   Mostly to improve the self logging
-   Is the benchmark properly thread-safe? Can guessChan be received in multiple places like that? Maybe should send a done signal down nextPasswordChan to the workers?
- Add new LAST_STASH_ENCRYPTION_KEY env var to allow STASH_ENCRYPTION_KEY to be rotated
-   Bump priority of jobs as they get older
- Change tests to use the test logger

# To watch

-   Timeouts sometimes incorrectly send 500s?

# Errors to investigate

-   11:42:03 ERR messengers/registry.go:328 failed to enqueue message send messengerType=discord_1 error="messengers [package] error: send error: enqueue job error: jobs [package] error: enqueue error: database [general] error: timeout [general] error: context deadline exceeded"
11:42:03 ERR loggers/loggers.go:497 failed to message admin about an error error="db common [package] error: WithTx error: commit transaction error: database [general] error: other error: sql: transaction has already been committed or rolled back"

-   00:07:52 ERR schedulers\delayFuncs.go:68 unable to create initial PeriodicTask object error="db common [package] error: WithTx error: start transaction error: database [general] error: other error: ent: starting a transaction: SQL logic error: cannot start a transaction within a transaction (1)" periodicTaskName=SEND_ACTIVE_SESSION_REMINDERS

- Clients cancelling requests causes internal errors?
15:44:34 ERR middleware/logging.go:52 an internal server error occurred url=/api/v1/users/get-authorization-code/ method=POST error="db common [package] error: WithTx error: context canceled" statusCode=-1

# To research

How password managers work. Maybe a zero trust system is actually possible?

Is it safe to hash an Argon2id hash with Argon2id?
https://blog.ircmaxell.com/2015/03/security-issue-combining-bcrypt-with.html

Can long passwords be used to DoS?

Can I zero sensitive memory on the frontend and backend?


Can I wake up a sleeping railway app by just having a separate cron service send an HTTP request over the internal network?

Maybe have the server save the time periodically and on shutdown? Then when it starts it runs through the cron jobs it missed? It probably shouldn't run the same jobs multiple times though

Prevent replay attacks on download endpoint with challenge-response system? Not really a concern because if the attacker is able to view the requests, chances are they have access to the responses too, which include the encrypted stashes. Probably means they've compromised the client, server or somehow the network. 

# Testing

- Expand download tests to cover self/admin locks
-   Create mock messenger
-   -   Register it multiple times in place of the actual ones to ensure the contacts are being passed correctly?
-   Continue fixing linting errors once golang ci v2 is working properly in VSCode
-   Race condition fuzzer that spams a bunch of endpoints
-   -   Would be run with the -race flag
-   -   In particular, test that spamming get-authorization-code with the correct password then updating the password invalidates all of the codes generated using the old password
- Fix tests around messaging admin on error, they were still passing when I forgot to load the UserMessenger edge, which caused the logic to think there were no messengers
-   Endpoints
-   -   Do they cancel their work if a request times out? Can encryption/decryption run in the background?
