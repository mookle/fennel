defmodule Cart.RuntimeConfigTest do
  # The tests change the process environment, so they run alone.
  use ExUnit.Case, async: false

  @runtime Path.expand("../../config/runtime.exs", __DIR__)
  @vars ["FENNEL_CART_PRODUCT_URL", "FENNEL_CART_PRODUCT_TOKEN"]

  setup do
    saved = Map.new(@vars, &{&1, System.get_env(&1)})

    on_exit(fn ->
      Enum.each(saved, fn
        {name, nil} -> System.delete_env(name)
        {name, value} -> System.put_env(name, value)
      end)
    end)
  end

  defp read(env, vars) do
    Enum.each(@vars, &System.delete_env/1)
    Enum.each(vars, fn {name, value} -> System.put_env(name, value) end)
    Config.Reader.read!(@runtime, env: env)
  end

  @both %{
    "FENNEL_CART_PRODUCT_URL" => "http://product:8080/v1",
    "FENNEL_CART_PRODUCT_TOKEN" => "a-token"
  }

  test "the release reads the product URL and token" do
    cart = read(:prod, @both)[:cart]

    assert cart[:product_url] == "http://product:8080/v1"
    assert cart[:product_token] == "a-token"
  end

  test "the test run needs neither variable" do
    cart = read(:test, %{})[:cart]

    refute Keyword.has_key?(cart, :product_url)
    refute Keyword.has_key?(cart, :product_token)
  end

  for name <- ["FENNEL_CART_PRODUCT_URL", "FENNEL_CART_PRODUCT_TOKEN"] do
    test "the release refuses to start with #{name} unset" do
      assert_raise RuntimeError, ~r/#{unquote(name)}/, fn ->
        read(:prod, Map.delete(@both, unquote(name)))
      end
    end

    test "the release refuses to start with #{name} empty" do
      assert_raise RuntimeError, ~r/#{unquote(name)}/, fn ->
        read(:prod, Map.put(@both, unquote(name), ""))
      end
    end

    for blank <- [" ", "   ", "\t\n"] do
      test "the release refuses to start with #{name} of only whitespace #{inspect(blank)}" do
        assert_raise RuntimeError, ~r/#{unquote(name)}/, fn ->
          read(:prod, Map.put(@both, unquote(name), unquote(blank)))
        end
      end
    end
  end
end
