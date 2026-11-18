defmodule Cart.MixProject do
  use Mix.Project

  def project do
    [
      app: :cart,
      version: "0.1.0",
      elixir: "~> 1.20",
      start_permanent: Mix.env() == :prod,
      deps: deps()
    ]
  end

  def application do
    [
      extra_applications: [:logger],
      mod: {Cart.Application, []}
    ]
  end

  defp deps do
    [
      {:bandit, "~> 1.15"},
      {:plug, "~> 1.20"}
    ]
  end
end
