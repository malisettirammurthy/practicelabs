docker build -t rammurthymalisetti/order-service:mq ./order-service; docker push rammurthymalisetti/order-service:mq
docker build -t rammurthymalisetti/notification-service:mq ./notification-service; docker push rammurthymalisetti/notification-service:mq
docker build -t rammurthymalisetti/analytics-service:mq ./analytics-service; docker push rammurthymalisetti/analytics-service:mq
