import Config

# The tests call the router directly, so the test run starts no listener and
# never contends for the port with a running container.
config :cart, server: config_env() != :test
