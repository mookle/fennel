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

  @uuidv7 ~r/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
  @sku_id "0190a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b"

  defp post_json(path, body \\ nil) do
    conn =
      case body do
        nil ->
          conn(:post, path)

        body ->
          conn(:post, path, JSON.encode!(body))
          |> put_req_header("content-type", "application/json")
      end

    Cart.Router.call(conn, @opts)
  end

  describe "POST /v1/carts" do
    test "returns 201 with a fixed lowercase UUIDv7 id" do
      conn = post_json("/v1/carts")

      assert conn.status == 201
      assert get_resp_header(conn, "content-type") == ["application/json"]
      assert %{"id" => id} = JSON.decode!(conn.resp_body)
      assert id =~ @uuidv7
      assert id == Cart.Router.cart_id()
    end

    test "returns the same id on every call" do
      first = JSON.decode!(post_json("/v1/carts").resp_body)
      second = JSON.decode!(post_json("/v1/carts").resp_body)

      assert first == second
    end
  end

  describe "POST /v1/carts/:id/items" do
    test "returns 201 and echoes sku_id and quantity" do
      conn =
        post_json("/v1/carts/#{Cart.Router.cart_id()}/items", %{sku_id: @sku_id, quantity: 2})

      assert conn.status == 201
      assert get_resp_header(conn, "content-type") == ["application/json"]
      assert JSON.decode!(conn.resp_body) == %{"sku_id" => @sku_id, "quantity" => 2}
    end

    test "echoes the request rather than fixed values" do
      other_sku = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"

      conn =
        post_json("/v1/carts/#{Cart.Router.cart_id()}/items", %{sku_id: other_sku, quantity: 7})

      assert JSON.decode!(conn.resp_body) == %{"sku_id" => other_sku, "quantity" => 7}
    end

    test "answers the same for a cart id that POST /v1/carts never returned" do
      conn =
        post_json("/v1/carts/5b6c7d8e-9f0a-4b1c-8d2e-3f4a5b6c7d8e/items", %{
          sku_id: @sku_id,
          quantity: 2
        })

      assert conn.status == 201
      assert JSON.decode!(conn.resp_body) == %{"sku_id" => @sku_id, "quantity" => 2}
    end
  end

  describe "POST /v1/carts/:id/checkout" do
    test "returns 202 with a fixed lowercase UUIDv7 purchase_id" do
      conn = post_json("/v1/carts/#{Cart.Router.cart_id()}/checkout")

      assert conn.status == 202
      assert get_resp_header(conn, "content-type") == ["application/json"]
      assert %{"purchase_id" => purchase_id} = JSON.decode!(conn.resp_body)
      assert purchase_id =~ @uuidv7
      assert purchase_id == Cart.Router.purchase_id()
    end

    test "returns the same purchase_id on every call, distinct from the cart id" do
      path = "/v1/carts/#{Cart.Router.cart_id()}/checkout"
      first = JSON.decode!(post_json(path).resp_body)
      second = JSON.decode!(post_json(path).resp_body)

      assert first == second
      assert first["purchase_id"] != Cart.Router.cart_id()
    end
  end

  test "an unknown path returns 404" do
    conn = conn(:get, "/nope") |> Cart.Router.call(@opts)

    assert conn.status == 404
  end
end
