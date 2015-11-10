Please bear in mind that this document is under heavy
construction and is likely to change often and greatly.
The beacon backend provides the following endpoints.

## Posting a Beacon

Use the following REST request to post a beacon.

```http
POST /beacon HTTP/1.1
Content-Type: multipart/form-data; boundary=793d63336

--793d63336
Content-Type: application/json
{
    "user": 24601,
    "text": "Who am I?",
    "latitude": 45.0,
    "longitude": 45.0
}

--793d63336
Content-Type: image/jpeg
<BINARY_IMAGE_DATA>
```

In response, you will receive the following.
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 525600,
    "user": 24601,
    "text": "Who am I?",
    "latitude": 45.0,
    "longitude": 45.0,
    "time": "Sat Nov. 7 21:13:33 CST 2015"
}
```
The response is not multi-part and will be identical
to the supplied JSON, but with a post ID and time
posted added.

Note that in the future, once users have been fully
implemented, a username will be returned along with
all user IDs.

## Retrieving a Beacon

Use the following REST request to retrieve a beacon and
all comments associated with it.

```http
GET /beacon/1 HTTP/1.1
```

The 1 above may be replaced by the id of any beacon.
You will receive the following response.

```http
HTTP/1.1 200 OK
Content-Type: multipart/form-data; boundary=793d63336

--793d63336
Content-Type: application/json
{
    "id": 1,
    "user": 24601,
    "text": "Who am I?",
    "hearts": 1,
    "latitude": 45.0,
    "longitude": 45.0,
    "time": "Sat Nov. 7 21:13:22 CST 2015",
    "comments": [
        {
            "id": 2,
            "user": 54321,
            "text": "This post is bad and you should feel bad.",
            "hearts": 7,
            "time": "Sat Nov. 7 21:13:33 CST 2015"
        },
        {
            "id": 3,
            "user": 12345,
            "text": "No, people. Let's be smart and bring it off.",
            "hearts": 5,
            "time": "Sat Nov. 7 21:13:57 CST 2015"
        }
    ]
}

--793d63336
Content-Type: image/jpeg
<BINARY_IMAGE_DATA>
```

## Hearting a Post

Send an empty POST to /heart/[post-id] to heart the corresponding
post.

```http
POST /heart/1 HTTP/1.1 
```

In response, you will receive a 200 OK if nothing has gone wrong.

```http
HTTP/1.1 200 OK
```

## Flagging a Post

The process of flagging a post is extremely similar to hearting
a post.
Send an empty POST to /flag/[post-id] to flag the corresponding
post.

```http
POST /flag/1 HTTP/1.1 
```

In response, you will receive a 200 OK if nothing has gone wrong.

```http
HTTP/1.1 200 OK
```

## General Errors
If any error condition is met while a request is being served, a
response similar to the following will be returned.

```http
HTTP/1.1 500 INTERNAL_SERVER_ERROR
Content-Type: application/json

{
    "error": "Could not retrieve post from db."
}
```
