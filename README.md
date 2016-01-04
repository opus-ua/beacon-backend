Beacon is a an open-source social network and accompanying Android app that
allows you to anonymously share interesting things and events with people in
your local community. Beacon is still in active development, but when finished,
using Beacon will be something like this:

Users take a picture in-app. Beacon takes a geotag and posts it on the map.
Beacons that are more popular are indicated on the map. You can select a beacon
to see what people are saying about it and to add your own comment.

A video demonstrating the current functionality can be found
[here](https://www.youtube.com/watch?v=KVeSS2WxJBo).

Binaries can be downloaded [here](http://bin.gnossen.com/beacon-backend/).

Beacon was started as a university project and as such, is not currently open
for pull requests. In January 2016, however, Beacon will become open for
contributions.

Please bear in mind that this document is under heavy
construction and is likely to change often and greatly.
The beacon backend provides the following endpoints.

## Configuration

The beacon backend requires an ID from Google in order to
integrate with Google sign-in. There are more detailed
instructions in the frontend's README on how to get such an
ID from Google. This ID should end in 
```apps.googleusercontent.com```. Take this ID and make it
the sole content of a new file ```google.id``` at the root
directory of the backend (no spaces or newlines). During the
build process, this ID will be incorporated into the binary
and used to verify new accounts.

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
    "id": 525600
}
```
The response is simply the id of the newly created beacon.

## Retrieving a Beacon

Use the following REST request to retrieve a beacon and
all comments associated with it.

```http
GET /beacon/1 HTTP/1.1
```

The 1 above may be replaced by the id of any beacon.
You will receive the following response.

Note that if you supply BasicAuth, the backend will
report which posts you've hearted. Otherwise, all
```hearted``` fields will be false.

```http
HTTP/1.1 200 OK
Content-Type: multipart/form-data; boundary=793d63336

--793d63336
Content-Type: application/json
{
    "id": 1,
    "userid": 24601,
    "username": "Jean Valjean",
    "text": "Who am I?",
    "hearts": 1,
    "latitude": 45.0,
    "longitude": 45.0,
    "time": 14780923409,
    "hearted": true,
    "comments": [
        {
            "id": 2,
            "user": 54321,
            "text": "This post is bad and you should feel bad.",
            "hearts": 7,
            "time": 14780923409,
            "hearted": false,
        },
        {
            "id": 3,
            "user": 12345,
            "text": "No, people. Let's be smart and bring it off.",
            "hearts": 5,
            "time": 14780923409,
            "hearted": false,
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

## Creating an account

Send a POST to /createaccount in order to create a new account.
Note that this is the only POST endpoint that does not require
BasicAuth.

```http
POST /createaccount
Content-Type: application/json

{
    "username": "dexter",
    "token": "....apps.googleusercontent.com",
}
```

You will receive a ``200 OK`` if the request is successful.
The request will fail if the username is taken or if the Google
account has already been used to open a Beacon account.

Along with the ```200 OK```, you will receive your user ID
and the secret. This pair will be used with http BasicAuth
for all future posts.

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 24601,
    "secret": "3SECRET5U"
}
```

## Getting Local Beacons
The endpoint ```/local``` is used to retrieve beacons posted within
a radius of a given set of GPS coordinates. To use it, simply use
the following HTTP. (note that BasicAuth is not required)

```http
POST /local
Content-Type: application/json

{
    "latitude": 33.219,
    "longitude": -87.544,
    "radius": 1.0
}
```

The radius should be given in miles. In response, you should get the
the following:

```http
HTTP/1.1 200 OK
Content-Type: multipart/form-data; boundary=793d63336

--793d63336
Content-Type: application/json

{
    beacons: [
        {
            "id": 1,
            "userid": 10,
            "text": "First ever post! Whoo!",
            "latitude": 33.218,
            "longitude": -87.544
        },
        {
            "id": 2,
            "userid": 11,
            "text": "Second ever post! Whoo!",
            "latitude": 33.219,
            "longitude": -87.543
        }
    ]
}

--793d63336
Content-Type: image/jpeg

<BINARY_IMAGE_DATA_FOR_POST_1>

--793d63336
Content-Type: image/jpeg

<BINARY_IMAGE_DATA_FOR_POST_2>
```

The order of images parts following the json is the same as the
order of posts within the json.

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
