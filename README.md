# Gophish 

edermi's custom gophish fork.
Don't expect this one to work for you. 
This fork does not work with regular phishing pages.
Please don't open issues, I'll ignore / close them. 
If you decide to use this, you're on your own.

## Changes compared to vanilla gophish

### Custom error page templating for 404 error pages in phishing handler

Allows to show a more benign 404 page. Also may be used for redirecting with some Javascript: `<script>window.location.replace("https://target.fqdn");</script>`

Changes made:
- Add `templates/404.html`
- In `controllers/phish.go`, add custom replacements for `http.NotFound` and `http.Error`

### Instead of showing phishing pages, request HTTP auth

HTTP auth is requested for users hitting their landing page. 
If an HTTP auth header is present, the data is extracted and stored as if the user had typed it into a login field.
After a user authenticates, he is redirected to the legit redirect URL.

Changes made:
- Add bluemonday HTML sanitizer lib (not necessary, but otherwise displaying a realm message without HTML tags is more work)
- `models/page.go` has a mandatory check for a redirect URL
- `controllers/phish.go` has modified `PhishHandler` and `renderPhishResponse` functions. The GET/POST logic is swapped with a "Authorization present, if not, request it" logic.