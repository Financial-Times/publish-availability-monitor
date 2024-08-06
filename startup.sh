#!/bin/sh
sed -i "s \"KAFKA_TOPIC\" \"$KAFKA_TOPIC\" " /config.json
sed -i "s \"KAFKA_CONSUMER_GROUP\" \"$KAFKA_CONSUMER_GROUP\" " /config.json
sed -i "s \"KAFKA_LAG_TOLERANCE\" $KAFKA_LAG_TOLERANCE " /config.json
sed -i "s \"KAFKA_ADDR\" \"$KAFKA_ADDR\" " /config.json
sed -i "s \"KAFKA_CLUSTER_ARN\" \"$KAFKA_CLUSTER_ARN\" " /config.json
sed -i "s \"CONTENT_URL\" \"$CONTENT_URL\" " /config.json
sed -i "s \"CONTENT_NEO4J_URL\" \"$CONTENT_NEO4J_URL\" " /config.json
sed -i "s \"CONTENT_COLLECTION_NEO4J_URL\" \"$CONTENT_COLLECTION_NEO4J_URL\" " /config.json
sed -i "s \"COMPLEMENTARY_CONTENT_URL\" \"$COMPLEMENTARY_CONTENT_URL\" " /config.json
sed -i "s \"LISTS_URL\" \"$LISTS_URL\" " /config.json
sed -i "s \"PAGES_URL\" \"$PAGES_URL\" " /config.json
sed -i "s \"CONTENT_RELATION_URL\" \"$CONTENT_RELATION_URL\" " /config.json
sed -i "s \"LIST_NOTIFICATIONS_URL\" \"$LIST_NOTIFICATIONS_URL\" " /config.json
sed -i "s \"PAGE_NOTIFICATIONS_URL\" \"$PAGE_NOTIFICATIONS_URL\" " /config.json
sed -i "s \"NOTIFICATIONS_PUSH_PUBLICATION_MONITOR_LIST\" \"$NOTIFICATIONS_PUSH_PUBLICATION_MONITOR_LIST\" " /config.json
sed -i "s \"NOTIFICATIONS_URL\" \"$NOTIFICATIONS_URL\" " /config.json
sed -i "s \"LIST_NOTIFICATIONS_PUSH_URL\" \"$LIST_NOTIFICATIONS_PUSH_URL\" " /config.json
sed -i "s \"PAGE_NOTIFICATIONS_PUSH_URL\" \"$PAGE_NOTIFICATIONS_PUSH_URL\" " /config.json
sed -i "s \"NOTIFICATIONS_PUSH_URL\" \"$NOTIFICATIONS_PUSH_URL\" " /config.json
sed -i "s \"LIST_NOTIFICATIONS_PUSH_API_KEY\" \"$LIST_NOTIFICATIONS_PUSH_API_KEY\" " /config.json
sed -i "s \"PAGE_NOTIFICATIONS_PUSH_API_KEY\" \"$PAGE_NOTIFICATIONS_PUSH_API_KEY\" " /config.json
sed -i "s \"NOTIFICATIONS_PUSH_API_KEY\" \"$NOTIFICATIONS_PUSH_API_KEY\" " /config.json
sed -i "s \"INTERNAL_COMPONENTS_URL\" \"$INTERNAL_COMPONENTS_URL\" " /config.json
sed -i "s \"VIDEO_MAPPER_URL\" \"$VIDEO_MAPPER_URL\" " /config.json
sed -i "s \"UPP_INTERNAL_ARTICLE_VALIDATOR_URL\" \"$UPP_INTERNAL_ARTICLE_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_LIST_VALIDATOR_URL\" \"$UPP_LIST_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_PAGE_VALIDATOR_URL\" \"$UPP_PAGE_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_INTERNAL_CPH_VALIDATOR_URL\" \"$UPP_INTERNAL_CPH_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_IMAGE_VALIDATOR_URL\" \"$UPP_IMAGE_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_IMAGE_SET_VALIDATOR_URL\" \"$UPP_IMAGE_SET_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_GRAPHIC_VALIDATOR_URL\" \"$UPP_GRAPHIC_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_CONTENT_COLLECTION_VALIDATOR_URL\" \"$UPP_CONTENT_COLLECTION_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_INTERNAL_LIVE_BLOG_PACKAGE_VALIDATOR_URL\" \"$UPP_INTERNAL_LIVE_BLOG_PACKAGE_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_INTERNAL_LIVE_BLOG_POST_VALIDATOR_URL\" \"$UPP_INTERNAL_LIVE_BLOG_POST_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_AUDIO_VALIDATOR_URL\" \"$UPP_AUDIO_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_CLIP_VALIDATOR_URL\" \"$UPP_CLIP_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_CLIP_SET_VALIDATOR_URL\" \"$UPP_CLIP_SET_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_CONTENT_RELATION_VALIDATOR_URL\" \"$UPP_CONTENT_RELATION_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_CUSTOM_CODE_COMPONENT_VALIDATOR_URL\" \"$UPP_CUSTOM_CODE_COMPONENT_VALIDATOR_URL\" " /config.json
sed -i "s \"GRAPHITE_ADDRESS\" \"$GRAPHITE_ADDRESS\" " /config.json
sed -i "s \"GRAPHITE_UUID\" \"$GRAPHITE_UUID\" " /config.json
sed -i "s \"ENVIRONMENT\" \"$ENVIRONMENT\" " /config.json
sed -i "s \"LOG_LEVEL\" \"$LOG_LEVEL\" " /config.json
sed -i "s/PUBLICATION_UUIDS/[$(echo "$PUBLICATION_UUIDS" | sed 's/[^,]\+/"&"/g' | sed 's/,/, /g')]/" /config.json

exec ./publish-availability-monitor -config /config.json
