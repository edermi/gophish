# Gophish 

edermi's custom gophish fork.
Don't expect this one to work for you. 
Please don't open issues, I'll ignore / close them. 
If you decide to use this, you're on your own.

## Changes compared to vanilla gophish

### Custom error page templating for 404 error pages in phishing handler

Allows to show a more benign 404 page. Also may be used for redirecting with some Javascript: `<script>window.location.replace("https://target.fqdn");</script>`

Changes made:
- Add `templates/404.html`
- In `controllers/phish.go`, add custom replacements for `http.NotFound` and `http.Error`

