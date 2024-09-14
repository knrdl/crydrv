# cryDrv

## encrypted CRUD everything web drive

- data encryption on rest
- CRUD REST API
- arbitrary accounts via HTTP Basic Auth
- client can read and write all filepaths for the current account
- user can build custom DIY webpages
- serve index.html for directories

## Protocol

1. Server: starts with env var `secret_key` being a (base64 encoded) 32-octets random key
2. Client: sends HTTP request. No basic auth or cookie provided => must authenticate
---
3. Client: sends basic auth with arbitrary `username` and 16+ chars `password`
4. Server: `userSalt = hkdf(secret_key, salt=username)`
5. Server: `userKey = argon2id(password, salt=userSalt)`
6. Server: `hashKey = hkdf(userKey, salt=userSalt)`
7. Server: create directory named `hashKey` (base64 encoded) if it doesn't exist
8. Server: attach Cookie with value `userKey` to Client
---
9. Client: GET file at `path` "/a/b.c"
10. Server: `filename = hkdf(userKey, salt=userSalt + path)`
11. Server: Serve file `hashKey` / `filename` under URL path `path` (if it exists in filesystem)
12. Client: POST/PUT file `content` at `path` "/a/b.c"
13. Server: calculates the filepath `hashKey` / `filename`, encrypts the file `file = aes256gcm(content, userKey, nonce)` and stores `file` under this path. `file` is encrypted chunkwise with a new `nonce` every 4MiB (and PKCS#7 padding)
14. Client: DELETE file at `path` "/a/b.c"
15. Server: calculate `hashKey` / `filename` and delete the file if it exists under this path
---
16. Client: uses the Cookie (see 8.) instead of basic auth. 
17. Server: calculate `hashKey` from provided `userKey` in cookie. The provided `userKey` is valid if the directory `hashKey` exists (see 7.)
---
18. Server-Admin: closes the registration
19. Client: can now only work with `username`/`password` combinations for which a `hashKey` directory exists on the server

## Threat model

- User has to trust the webserver blindly (as with all web apps)
- Webserver doesn't have to trust the storage/backup provider (e.g. cloud)
- Storage provider can still see file count & sizes => knows the number of users, can guess possible file contents
- Storage provider still sees file metadata => can track (anonymous) user activities via timestamps
- add HTTPS for transport encryption

## Setup

```yaml
services:
  crydrv:
    image: ghcr.io/knrdl/crydrv:edge
    restart: always
    environment:
      - OPEN_REGISTRATION=true
    # - SECRET_KEY=...  # generated on first start
    ports:
      - 8000:8000
    volumes:
      - ./data:/www  # chown -R 1000:1000 ./data

    mem_limit: 4g
    memswap_limit: 4g
```

getting started: see [webutils](./webutils)