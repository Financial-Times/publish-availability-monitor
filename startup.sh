#!/bin/sh
sed -i "s \"KAFKA_TOPIC\" \"$KAFKA_TOPIC\" " /config.json
sed -i "s \"KAFKA_PROXY_HOST\" \"$KAFKA_PROXY_HOST\" " /config.json
sed -i "s \"QUEUE_ADDR\" \"$QUEUE_ADDR\" " /config.json
sed -i "s \"CONTENT_URL\" \"$CONTENT_URL\" " /config.json
sed -i "s \"CONTENT_NEO4J_URL\" \"$CONTENT_NEO4J_URL\" " /config.json
sed -i "s \"COMPLEMENTARY_CONTENT_URL\" \"$COMPLEMENTARY_CONTENT_URL\" " /config.json
sed -i "s \"LISTS_URL\" \"$LISTS_URL\" " /config.json
sed -i "s \"GENERIC_LISTS_URL\" \"$GENERIC_LISTS_URL\" " /config.json
sed -i "s \"LISTS_NOTIFICATIONS_URL\" \"$LISTS_NOTIFICATIONS_URL\" " /config.json
sed -i "s \"NOTIFICATIONS_URL\" \"$NOTIFICATIONS_URL\" " /config.json
sed -i "s \"LISTS_NOTIFICATIONS_PUSH_URL\" \"$LISTS_NOTIFICATIONS_PUSH_URL\" " /config.json
sed -i "s \"NOTIFICATIONS_PUSH_URL\" \"$NOTIFICATIONS_PUSH_URL\" " /config.json
sed -i "s \"LISTS_NOTIFICATIONS_PUSH_API_KEY\" \"$LISTS_NOTIFICATIONS_PUSH_API_KEY\" " /config.json
sed -i "s \"NOTIFICATIONS_PUSH_API_KEY\" \"$NOTIFICATIONS_PUSH_API_KEY\" " /config.json
sed -i "s \"INTERNAL_COMPONENTS_URL\" \"$INTERNAL_COMPONENTS_URL\" " /config.json
sed -i "s \"METHODE_ARTICLE_VALIDATION_URL\" \"$METHODE_ARTICLE_VALIDATION_URL\" " /config.json
sed -i "s \"METHODE_CONTENT_PLACEHOLDER_MAPPER_URL\" \"$METHODE_CONTENT_PLACEHOLDER_MAPPER_URL\" " /config.json
sed -i "s \"METHODE_LIST_VALIDATION_URL\" \"$METHODE_LIST_VALIDATION_URL\" " /config.json
sed -i "s \"METHODE_IMAGE_MODEL_MAPPER_URL\" \"$METHODE_IMAGE_MODEL_MAPPER_URL\" " /config.json
sed -i "s \"METHODE_ARTICLE_INTERNAL_COMPONENTS_MAPPER_URL\" \"$METHODE_ARTICLE_INTERNAL_COMPONENTS_MAPPER_URL\" " /config.json
sed -i "s \"VIDEO_MAPPER_URL\" \"$VIDEO_MAPPER_URL\" " /config.json
sed -i "s \"WORDPRESS_MAPPER_URL\" \"$WORDPRESS_MAPPER_URL\" " /config.json
sed -i "s \"UPP_INTERNAL_ARTICLE_VALIDATOR_URL\" \"$UPP_INTERNAL_ARTICLE_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_LIST_VALIDATOR_URL\" \"$UPP_LIST_VALIDATOR_URL\" " /config.json
sed -i "s \"UPP_INTERNAL_CPH_VALIDATOR_URL\" \"$UPP_INTERNAL_CPH_VALIDATOR_URL\" " /config.json
sed -i "s \"UUID_RESOLVER_URL\" \"$UUID_RESOLVER_URL\" " /config.json
sed -i "s \"GRAPHITE_ADDRESS\" \"$GRAPHITE_ADDRESS\" " /config.json
sed -i "s \"GRAPHITE_UUID\" \"$GRAPHITE_UUID\" " /config.json
sed -i "s \"ENVIRONMENT\" \"$ENVIRONMENT\" " /config.json

exec ./publish-availability-monitor -config /config.json
