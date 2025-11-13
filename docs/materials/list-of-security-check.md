| Name                                    | Category                              | Implemented |
| --------------------------------------- | ------------------------------------- | ----------- |
| Frame Security Policy (X-Frame-Options) | Clickjacking Protection               | ✅          |
| Access-Control-Allow-Credentials header | Cross-Origin Resource Sharing (CORS)  | ✅          |
| Access-Control-Allow-Headers header     | Cross-Origin Resource Sharing (CORS)  | ⛔️          |
| Access-Control-Allow-Origin header      | Cross-Origin Resource Sharing (CORS)  | ✅          |
| Access-Control-Expose-Headers header    | Cross-Origin Resource Sharing (CORS)  | ⛔️          |
| Access-Control-Max-Age header           | Cross-Origin Resource Sharing (CORS)  | ⛔️          |
| Cross-Origin-Embedder-Policy header     | Cross-Origin Resource Sharing (CORS)  | ✅          |
| Cross-Origin-Opener-Policy header       | Cross-Origin Resource Sharing (CORS)  | ✅          |
| Cross-Origin-Resource-Policy header     | Cross-Origin Resource Sharing (CORS)  | ⛔️          |
| Cross-Origin Resource Isolation         | Cross-Origin Resource Sharing (CORS)  | ✅ (via COOP/COEP) |
| Vary: Origin header (CORS caching)      | Cross-Origin Resource Sharing (CORS)  | ⛔️          |
| Content Security Policy (CSP)           | Content Security Policy (CSP)         | ✅          |
| Content Security Policy (CSP) Bypass    | Content Security Policy (CSP)         | ⛔️          |
| Set-Cookie headers (Secure/HttpOnly)    | Cookie Security                       | ✅          |
| Open Ports                              | Network Security                      | ⛔️          |
| Subdomain Takeover                      | Network Security                      | ⛔️          |
| Permissions-Policy header               | Miscellaneous Headers                 | ✅          |
| Referrer Policy                         | Miscellaneous Headers                 | ✅          |
| Server information disclosure           | Miscellaneous Headers                 | ✅          |
| Content-Type header                     | Miscellaneous Headers                 | ⛔️          |
| Deprecated X-XSS-Protection header      | Miscellaneous Headers                 | ✅          |
| Vulnerable JS Libraries                 | Miscellaneous                         | ⛔️          |
| Anti-CSRF Tokens                        | Cross-Site Scripting (XSS) Protection | ⛔️          |
| Trusted Types readiness                 | Cross-Site Scripting (XSS) Protection | ⛔️          |
| X-Content-Type-Options                  | Cross-Site Scripting (XSS) Protection | ✅          |
| Certificate Hostname & Chain            | Transport Layer Security (TLS)        | ✅          |
| Certificate Expiry                      | Transport Layer Security (TLS)        | ✅          |
| Cipher Suite                            | Transport Layer Security (TLS)        | ✅          |
| Deprecated TLS versions supported       | Transport Layer Security (TLS)        | ✅          |
| HTTPS enabled                           | Transport Layer Security (TLS)        | ⛔️ (implicit only) |
| HSTS enabled                            | Transport Layer Security (TLS)        | ✅ (via header) |
| Mixed Content                           | Transport Layer Security (TLS)        | ⛔️          |
| OCSP Stapling                           | Transport Layer Security (TLS)        | ⛔️          |
| TLS Version                             | Transport Layer Security (TLS)        | ✅          |
