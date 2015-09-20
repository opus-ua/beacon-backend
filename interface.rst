# Beacon REST Interface

## Authentication
At the moment, we have not hammered out the authentication method, but when
implemented, all requests will require an authorization line in the HTTP header.
Note that this REST interface operates strictly over HTTPS.

## Posting an Image
Request:

```http
POST /post HTTP/1.1
Content-Type: multipart/mixed; boundary=frontier

--frontier
Content-Type: application/json

{
    "loc": <geotag>,
    "desc": <string>
}

--frontier
Content-Type: image/jpeg

...

```

Response:
```http
HTTP/1.1 201 CREATED
Content-Type: application/json

{
    "post-id": <string>
}
```

If the operation is unsuccessful, an error code such as 501 will be returned.
