defmodule Cart.Router do
  @moduledoc """
  The HTTP routes of the `cart` service.

  `/healthz` sits outside `/v1` and takes no auth, so Compose and an operator
  can probe the service without a token.
  """

  use Plug.Router

  plug(:match)
  plug(:dispatch)

  get "/healthz" do
    # JSON defines no charset parameter (RFC 8259), so the header carries none.
    conn
    |> put_resp_content_type("application/json", nil)
    |> send_resp(200, JSON.encode!(%{status: "ok"}))
  end

  match _ do
    send_resp(conn, 404, "")
  end
end
