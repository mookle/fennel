defmodule Cart.Router do
  @moduledoc """
  The HTTP routes of the `cart` service.

  `/healthz` sits outside `/v1` and takes no auth, so Compose and an operator
  can probe the service without a token.

  The `/v1` routes are a facade at this stage. They hold no state and validate
  no input. Creating a Cart always returns the same cart id, adding a CartItem
  echoes the request's `sku_id` and `quantity`, and checkout always returns the
  same `purchase_id`. `contracts/cart.openapi.yaml` uses the same values in its
  examples.
  """

  use Plug.Router

  # The fixed ids the facade returns. Both are lowercase UUIDv7 strings
  # (ADR-0024).
  @cart_id "019e8a3c-5d20-7a41-9c3e-2b7f1d6e4a80"
  @purchase_id "019e8a3c-7f14-7b92-8d05-6c1a9e3f2b57"

  plug(:match)
  plug(Plug.Parsers, parsers: [:json], pass: ["*/*"], json_decoder: JSON)
  plug(:dispatch)

  @doc "The cart id that `POST /v1/carts` returns."
  def cart_id, do: @cart_id

  @doc "The purchase id that `POST /v1/carts/{id}/checkout` returns."
  def purchase_id, do: @purchase_id

  get "/healthz" do
    send_json(conn, 200, %{status: "ok"})
  end

  post "/v1/carts" do
    send_json(conn, 201, %{id: @cart_id})
  end

  post "/v1/carts/:_id/items" do
    item = Map.take(conn.body_params, ["sku_id", "quantity"])
    send_json(conn, 201, item)
  end

  post "/v1/carts/:_id/checkout" do
    send_json(conn, 202, %{purchase_id: @purchase_id})
  end

  match _ do
    send_resp(conn, 404, "")
  end

  defp send_json(conn, status, body) do
    # JSON defines no charset parameter (RFC 8259), so the header carries none.
    conn
    |> put_resp_content_type("application/json", nil)
    |> send_resp(status, JSON.encode!(body))
  end
end
