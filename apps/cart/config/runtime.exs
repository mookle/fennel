import Config

# Read at start, not at compile time, so one release image serves on any port.
config :cart, port: String.to_integer(System.get_env("FENNEL_CART_PORT", "4000"))

# The release refuses to start without the URL of product and the token cart
# presents to it. Only the release reads them, so mix test needs neither. A
# value of only whitespace counts as empty.
if config_env() == :prod do
  required = fn name ->
    value = System.get_env(name)

    if value == nil or String.trim(value) == "" do
      raise "environment variable #{name} is missing or empty; the release cannot start without it"
    end

    value
  end

  config :cart,
    product_url: required.("FENNEL_CART_PRODUCT_URL"),
    product_token: required.("FENNEL_CART_PRODUCT_TOKEN")
end
