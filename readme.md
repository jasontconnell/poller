# Poller

Poll defined urls and report status on http://localhost:4444/status (by default)

### Sample Configuration

(see config.json)

### sample requests file

GET requests need 3 tokens, POST requests need 5, with content type and body added

```
POST site1 /-/some/api application/json {}
GET site1 /index.html
GET site2 /index.html
POST site2 /-/some/other application/json { "foo": 4 }
```