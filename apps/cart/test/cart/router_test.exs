defmodule Cart.RouterTest do
  use ExUnit.Case, async: true

  import Plug.Test
  import Plug.Conn

  @opts Cart.Router.init([])

  test "GET /healthz returns 200 with a JSON status" do
    conn = conn(:get, "/healthz") |> Cart.Router.call(@opts)

    assert conn.status == 200
    assert get_resp_header(conn, "content-type") == ["application/json"]
    assert JSON.decode!(conn.resp_body) == %{"status" => "ok"}
  end

  test "GET /healthz needs no Authorization header" do
    conn = conn(:get, "/healthz") |> Cart.Router.call(@opts)

    assert conn.status == 200
  end

  test "an unknown path returns 404" do
    conn = conn(:get, "/nope") |> Cart.Router.call(@opts)

    assert conn.status == 404
  end
end
