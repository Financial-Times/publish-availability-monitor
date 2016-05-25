#!/bin/sh
sed -i "s QUEUE_ADDR $QUEUE_ADDR " /config.json
sed -i "s ENVIRONMENT $ENVIRONMENT " /config.json
sed -i "s S3_URL $S3_URL " /config.json
sed -i "s CONTENT_URL $CONTENT_URL " /config.json
sed -i "s LISTS_URL $LISTS_URL " /config.json
sed -i "s NOTIFICATIONS_URL $NOTIFICATIONS_URL " /config.json

exec ./app -config /config.json
