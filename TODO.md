# TODO

- Create endpoint for creating a user with a given username. Can then generate a signup link for a user to update their stash and password. That way the user doesn't need to give the admin their password and data, and the admin doesn't need to give their password/download session
- Encrypt stashes using a random key and store that key encrypted with the user's password hash? Instead of encrypting with user's password hash directly. Would that prevent side channel attacks from revealing stash size? Maybe makes it more flexible in the future?
- Hash authorisation codes with SHA256 to ensure read access to the database doesn't allow active download sessions to be hijacked
- Move env encryption from the service? Stash content and filenames don't need to be encrypted by it because the encryption keys for them are encrypted with the env var
- Allow creating signup links to change stash contents/password
- Use "Cache-Control": "no-store" on sensitive endpoints?
- Disk usage keeps increasing. Maybe need to delete old job executions and logs? Implement the dump database endpoint so I can inspect
-   Improve frontend
-   Remove userID and publicMessage from logger, it's not worth the complexity and risks
- Encrypt stashes with an extra key to prevent offline attacks if database is leaked
-   00:07:52 ERR schedulers\delayFuncs.go:68 unable to create initial PeriodicTask object error="db common [package] error: WithTx error: start transaction error: database [general] error: other error: ent: starting a transaction: SQL logic error: cannot start a transaction within a transaction (1)" periodicTaskName=SEND_ACTIVE_SESSION_REMINDERS
-   Can cancelling requests make views non-atomic if a view uses multiple transactions? Are there any security risks with this?
-   Standardise returning errors and using gin.H vs the endpoint specific download struct. That struct applies defaults which the other 2 approaches don't, so it could leak information
-   Experiment using Cloudflare to prevent DDoS requests on the hashing endpoint. It's not a great idea to shift the hashing to the client due to WASM and different devices' RAM limitations. Can specifically limit that endpoint
-   Limit number of concurrent hash requests to avoid using too much RAM
-   Avoid sending successful responses inside a transaction because it could fail while committing?
-   Add limits on self-locking so a hacker can't lock you out forever
-   -   Attempting to get an authorisation code when locked should send the unlock date
-   Repeat password in sign up form
-   -   Admins should be able to reset it so if there's an unauthorised login, the user can block with a self lock, the admin can reset them and then they can block again without waiting
- Implement more messengers:
- - Email (SendGrid)
- - ntfy.sh
- - Matrix?
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
- Standardise error handling on the frontend

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

# To research

How password managers work. Maybe a zero trust system is actually possible?

Can I wake up a sleeping railway app by just having a separate cron service send an HTTP request over the internal network?

Maybe have the server save the time periodically and on shutdown? Then when it starts it runs through the cron jobs it missed? It probably shouldn't run the same jobs multiple times though

Reduce the necessary trust in the server by using OPAQUE?
https://blog.cloudflare.com/opaque-oblivious-passwords/

# Testing

-   Create mock messenger
-   -   Register it multiple times in place of the actual ones to ensure the contacts are being passed correctly?
-   Continue fixing linting errors once golang ci v2 is working properly in VSCode
-   Race condition fuzzer that spams a bunch of endpoints
-   -   Would be run with the -race flag
-   -   In particular, test that spamming get-authorization-code with the correct password then updating the password invalidates all of the codes generated using the old password
-   Endpoints
-   -   Do they cancel their work if a request times out? Can encryption/decryption run in the background?
