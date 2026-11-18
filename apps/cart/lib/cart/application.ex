defmodule Cart.Application do
  @moduledoc false

  use Application

  @impl true
  def start(_type, _args) do
    opts = [strategy: :one_for_one, name: Cart.Supervisor]
    Supervisor.start_link(children(), opts)
  end

  defp children do
    if Application.fetch_env!(:cart, :server) do
      [{Bandit, plug: Cart.Router, port: Application.fetch_env!(:cart, :port)}]
    else
      []
    end
  end
end
