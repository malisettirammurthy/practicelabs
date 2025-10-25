from prometheus_client import start_http_server, Counter, Gauge
import random, time

# Define metrics
REQUEST_COUNT = Counter("http_requests_total", "Total HTTP requests served")
TEMPERATURE = Gauge("room_temperature_celsius", "Simulated room temperature in Celsius")
RAM_TEST_METRIC_COUNT = Counter("ram_test_metric_count", "Ram's sample custom metric")

def generate_metrics():
    while True:
        REQUEST_COUNT.inc()                       # increment request counter
        RAM_TEST_METRIC_COUNT.inc()
        TEMPERATURE.set(random.uniform(20, 35))    # random temperature value
        time.sleep(2)                              # update every 2 seconds

if __name__ == "__main__":
    start_http_server(8080)   # exposes /metrics on port 8080
    print("Serving metrics on :8080/metrics")
    generate_metrics()
