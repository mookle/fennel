import Config

# Read at start, not at compile time, so one release image serves on any port.
config :cart, port: String.to_integer(System.get_env("FENNEL_CART_PORT", "4000"))
