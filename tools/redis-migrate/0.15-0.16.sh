#!/bin/bash

# Changes to Redis configuration
#   * Store Times as base 10 integers representing a Unix timestamp
#   * Update thumbnail size from 100x150 to 200x300
#
# Goal: Migration script *must* be idempotent. ie, if this script is
#       run on an 0.16-compliant db, then the db will not be changed at all

if [ -z "$(which convert)" ]; then
    echo "Imagemagick must be installed."
    exit 1
fi

POSTS=$(redis-cli keys "p:*" | awk "{print $1}" | egrep "p:[0-9]+$")
for POST in $POSTS
do
    # Update timestamp
    TIME=$(redis-cli hget $POST time)

    if ! [[ $TIME =~ ^[0-9]+$ ]]; then
        TIME=$(date --date="$TIME" +"%s")
        redis-cli hset $POST time $TIME
    fi

    # Resize Thumbnail
    TYPE=$(redis-cli hget $POST type)
    if [ $TYPE == "beacon" ]; then
        redis-cli hget $POST img > /tmp/beacon-post-img.jpg
        convert /tmp/beacon-post-img.jpg -resize 200x300 /tmp/beacon-post-thumb.jpg
        redis-cli -x hset $POST thumb < /tmp/beacon-post-thumb.jpg
    fi
done
